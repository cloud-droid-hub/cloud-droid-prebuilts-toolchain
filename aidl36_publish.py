#!/usr/bin/env python3
# Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

import ctypes
import errno
import fcntl
import hashlib
import os
import secrets
import signal
import stat
import sys


RENAME_NOREPLACE = 1
RENAME_EXCHANGE = 2


class TxError(Exception):
    def __init__(self, code, message):
        super().__init__(message)
        self.code = code


pending_signal = 0


def on_signal(number, _frame):
    global pending_signal
    if pending_signal == 0:
        pending_signal = number


def abort_signal():
    if pending_signal:
        raise TxError(signal_code(), "publication interrupted before commit")


def signal_code():
    return 128 + pending_signal if pending_signal else 0


def same_id(info, expected):
    return (info.st_dev, info.st_ino) == expected


def stamp(info):
    return (
        info.st_dev,
        info.st_ino,
        info.st_mode,
        info.st_nlink,
        info.st_uid,
        info.st_gid,
        info.st_size,
        info.st_mtime_ns,
        info.st_ctime_ns,
    )


def parse_id(value):
    fields = value.split(":")
    if len(fields) < 2 or any(not field.isdigit() for field in fields[:2]):
        raise ValueError("invalid directory identity")
    return int(fields[0]), int(fields[1])


def dir_flags():
    return os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW | os.O_CLOEXEC


def check_name(value):
    if not value or value in (".", "..") or "/" in value or "\0" in value:
        raise ValueError("invalid publication name")


def rename2(old_fd, old, new_fd, new, flags):
    libc = ctypes.CDLL(None, use_errno=True)
    function = libc.renameat2
    function.argtypes = [ctypes.c_int, ctypes.c_char_p, ctypes.c_int, ctypes.c_char_p, ctypes.c_uint]
    function.restype = ctypes.c_int
    while function(old_fd, os.fsencode(old), new_fd, os.fsencode(new), flags) != 0:
        error = ctypes.get_errno()
        if error == errno.EINTR:
            continue
        raise OSError(error, os.strerror(error), new)
    return None


def hash_file(parent_fd, name, before, digest):
    descriptor = os.open(name, os.O_RDONLY | os.O_NOFOLLOW | os.O_CLOEXEC, dir_fd=parent_fd)
    try:
        opened = os.fstat(descriptor)
        if stamp(opened) != stamp(before):
            raise RuntimeError("publication file identity changed")
        digest.update(
            b"F\0"
            + format(stat.S_IMODE(opened.st_mode), "o").encode()
            + b"\0"
            + str(opened.st_uid).encode()
            + b"\0"
            + str(opened.st_gid).encode()
            + b"\0"
            + str(opened.st_size).encode()
            + b"\0"
            + str(opened.st_mtime_ns).encode()
            + b"\0"
            + str(opened.st_ctime_ns).encode()
            + b"\0"
        )
        while True:
            block = os.read(descriptor, 1024 * 1024)
            if not block:
                break
            digest.update(block)
        after_fd = os.fstat(descriptor)
        after = os.stat(name, dir_fd=parent_fd, follow_symlinks=False)
        if stamp(after_fd) != stamp(opened) or stamp(after) != stamp(opened):
            raise RuntimeError("publication file changed while hashing")
    finally:
        os.close(descriptor)


def hash_dir(descriptor, prefix, digest):
    before = os.fstat(descriptor)
    names = sorted(os.listdir(descriptor))
    digest.update(
        b"D\0"
        + prefix.encode()
        + b"\0"
        + format(stat.S_IMODE(before.st_mode), "o").encode()
        + b"\0"
        + str(before.st_uid).encode()
        + b"\0"
        + str(before.st_gid).encode()
        + b"\0"
        + str(before.st_size).encode()
        + b"\0"
        + str(before.st_mtime_ns).encode()
        + b"\0"
    )
    if prefix:
        digest.update(str(before.st_ctime_ns).encode() + b"\0")
    entries = []
    for name in names:
        if "\0" in name or "/" in name:
            raise RuntimeError("invalid publication entry")
        entry = os.stat(name, dir_fd=descriptor, follow_symlinks=False)
        entries.append((name, stamp(entry)))
        relative = name if not prefix else prefix + "/" + name
        mode = stat.S_IMODE(entry.st_mode)
        if stat.S_ISDIR(entry.st_mode):
            child = os.open(name, dir_flags(), dir_fd=descriptor)
            try:
                if not same_id(os.fstat(child), (entry.st_dev, entry.st_ino)):
                    raise RuntimeError("publication directory identity changed")
                hash_dir(child, relative, digest)
                after = os.stat(name, dir_fd=descriptor, follow_symlinks=False)
                if stamp(after) != stamp(entry):
                    raise RuntimeError("publication directory changed while hashing")
            finally:
                os.close(child)
        elif stat.S_ISREG(entry.st_mode):
            digest.update(relative.encode() + b"\0")
            hash_file(descriptor, name, entry, digest)
        else:
            raise RuntimeError("unsupported publication entry")
    for name, expected in entries:
        current = os.stat(name, dir_fd=descriptor, follow_symlinks=False)
        if stamp(current) != expected:
            raise RuntimeError("publication entry changed while hashing")
    after = os.fstat(descriptor)
    if stamp(after) != stamp(before) or sorted(os.listdir(descriptor)) != names:
        raise RuntimeError("publication directory changed while hashing")


def seal_fd(descriptor):
    info = os.fstat(descriptor)
    if not stat.S_ISDIR(info.st_mode):
        raise RuntimeError("publication root is not a directory")
    digest = hashlib.sha256()
    hash_dir(descriptor, "", digest)
    return f"{info.st_dev}:{info.st_ino}:{digest.hexdigest()}"


def seal_at(parent_fd, name):
    descriptor = os.open(name, dir_flags(), dir_fd=parent_fd)
    try:
        return seal_fd(descriptor)
    finally:
        os.close(descriptor)


def seal_path(path):
    descriptor = os.open(path, dir_flags())
    try:
        print(seal_fd(descriptor))
    finally:
        os.close(descriptor)


def check_parent(path, descriptor, expected):
    opened = os.fstat(descriptor)
    visible = os.stat(path, follow_symlinks=False)
    return stat.S_ISDIR(visible.st_mode) and same_id(opened, expected) and same_id(visible, expected)


def make_tx(parent_fd, stem):
    for _ in range(32):
        name = f".{stem}.tx.{secrets.token_hex(8)}"
        try:
            os.mkdir(name, 0o700, dir_fd=parent_fd)
        except FileExistsError:
            continue
        descriptor = -1
        try:
            descriptor = os.open(name, dir_flags(), dir_fd=parent_fd)
            os.fchmod(descriptor, 0o700)
            opened = os.fstat(descriptor)
            visible = os.stat(name, dir_fd=parent_fd, follow_symlinks=False)
            if stamp(opened) != stamp(visible):
                raise RuntimeError("publication transaction directory changed")
            os.fsync(parent_fd)
            return name, descriptor
        except Exception:
            if descriptor != -1:
                os.close(descriptor)
            try:
                os.rmdir(name, dir_fd=parent_fd)
            except OSError:
                pass
            raise
    raise RuntimeError("publication transaction directory allocation failed")


def drop_tx(parent_fd, name, descriptor):
    if os.listdir(descriptor):
        return False
    opened = os.fstat(descriptor)
    visible = os.stat(name, dir_fd=parent_fd, follow_symlinks=False)
    if stamp(opened) != stamp(visible):
        return False
    os.rmdir(name, dir_fd=parent_fd)
    os.fsync(parent_fd)
    return True


def move_back(parent_fd, tx_fd, name, public, expected_fd):
    current = os.stat(name, dir_fd=tx_fd, follow_symlinks=False)
    if expected_fd != -1:
        expected = os.fstat(expected_fd)
        if not same_id(current, (expected.st_dev, expected.st_ino)):
            return ""
    target = public
    try:
        rename2(tx_fd, name, parent_fd, target, RENAME_NOREPLACE)
    except FileExistsError:
        target = f".{public}.recover.{secrets.token_hex(8)}"
        rename2(tx_fd, name, parent_fd, target, RENAME_NOREPLACE)
    os.fsync(tx_fd)
    os.fsync(parent_fd)
    visible = os.stat(target, dir_fd=parent_fd, follow_symlinks=False)
    if expected_fd != -1:
        expected = os.fstat(expected_fd)
        if not same_id(visible, (expected.st_dev, expected.st_ino)):
            return ""
    return target


def check_tree(descriptor, root_dev):
    before = os.fstat(descriptor)
    if before.st_dev != root_dev:
        raise RuntimeError("cleanup tree crosses a filesystem boundary")
    names = sorted(os.listdir(descriptor))
    entries = []
    for name in names:
        entry = os.stat(name, dir_fd=descriptor, follow_symlinks=False)
        if entry.st_dev != root_dev:
            raise RuntimeError("cleanup tree crosses a filesystem boundary")
        child_plan = None
        if stat.S_ISDIR(entry.st_mode):
            child = os.open(name, dir_flags(), dir_fd=descriptor)
            try:
                child_info = os.fstat(child)
                if child_info.st_dev != root_dev:
                    raise RuntimeError("cleanup tree crosses a filesystem boundary")
                if not same_id(child_info, (entry.st_dev, entry.st_ino)):
                    raise RuntimeError("cleanup directory identity changed")
                child_plan = check_tree(child, root_dev)
                current = os.stat(name, dir_fd=descriptor, follow_symlinks=False)
                if stamp(current) != stamp(entry):
                    raise RuntimeError("cleanup directory changed during preflight")
            finally:
                os.close(child)
        elif not stat.S_ISREG(entry.st_mode):
            raise RuntimeError("unsupported cleanup entry")
        entries.append((name, stamp(entry), child_plan))
    for name, expected, _ in entries:
        current = os.stat(name, dir_fd=descriptor, follow_symlinks=False)
        if stamp(current) != expected:
            raise RuntimeError("cleanup entry changed during preflight")
    if stamp(os.fstat(descriptor)) != stamp(before) or sorted(os.listdir(descriptor)) != names:
        raise RuntimeError("cleanup directory changed during preflight")
    return entries


def hide_entry(parent_fd, name):
    for _ in range(32):
        hidden = f".clean.{secrets.token_hex(8)}"
        try:
            rename2(parent_fd, name, parent_fd, hidden, RENAME_NOREPLACE)
        except FileExistsError:
            continue
        os.fsync(parent_fd)
        return hidden
    raise RuntimeError("cleanup private name allocation failed")


def remove_tree(descriptor, root_dev, entries):
    names = [entry[0] for entry in entries]
    if sorted(os.listdir(descriptor)) != names:
        raise RuntimeError("cleanup directory changed after preflight")
    os.fchmod(descriptor, 0o700)
    for name, expected, child_plan in entries:
        before = os.stat(name, dir_fd=descriptor, follow_symlinks=False)
        if stamp(before) != expected:
            raise RuntimeError("cleanup entry changed after preflight")
        if before.st_dev != root_dev:
            raise RuntimeError("cleanup tree crosses a filesystem boundary")
        if stat.S_ISDIR(before.st_mode):
            child = os.open(name, dir_flags(), dir_fd=descriptor)
            try:
                if not same_id(os.fstat(child), (before.st_dev, before.st_ino)):
                    raise RuntimeError("cleanup directory identity changed")
                hidden = hide_entry(descriptor, name)
                current = os.stat(hidden, dir_fd=descriptor, follow_symlinks=False)
                if not same_id(current, (before.st_dev, before.st_ino)):
                    raise RuntimeError("cleanup directory changed during isolation")
                remove_tree(child, root_dev, child_plan)
                after_fd = os.fstat(child)
                after = os.stat(hidden, dir_fd=descriptor, follow_symlinks=False)
                if stamp(after) != stamp(after_fd):
                    raise RuntimeError("cleanup directory path changed")
                latest = os.stat(hidden, dir_fd=descriptor, follow_symlinks=False)
                if stamp(latest) != stamp(after_fd):
                    raise RuntimeError("cleanup directory path changed before removal")
                os.rmdir(hidden, dir_fd=descriptor)
                if not same_id(os.fstat(child), (before.st_dev, before.st_ino)):
                    raise RuntimeError("cleanup directory changed after removal")
            finally:
                os.close(child)
        elif stat.S_ISREG(before.st_mode):
            child = os.open(name, os.O_RDONLY | os.O_NOFOLLOW | os.O_CLOEXEC, dir_fd=descriptor)
            try:
                if not same_id(os.fstat(child), (before.st_dev, before.st_ino)):
                    raise RuntimeError("cleanup file identity changed")
                hidden = hide_entry(descriptor, name)
                current = os.stat(hidden, dir_fd=descriptor, follow_symlinks=False)
                if not same_id(current, (before.st_dev, before.st_ino)):
                    raise RuntimeError("cleanup file changed during isolation")
                os.unlink(hidden, dir_fd=descriptor)
                if not same_id(os.fstat(child), (before.st_dev, before.st_ino)):
                    raise RuntimeError("cleanup file changed after removal")
            finally:
                os.close(child)
        else:
            raise RuntimeError("unsupported cleanup entry")
    os.fsync(descriptor)


def remove_at(parent_fd, name, expected):
    descriptor = os.open(name, dir_flags(), dir_fd=parent_fd)
    try:
        if seal_fd(descriptor) != expected:
            raise RuntimeError("cleanup tree does not belong to this transaction")
        root_dev = os.fstat(descriptor).st_dev
        tree_plan = check_tree(descriptor, root_dev)
        hidden = hide_entry(parent_fd, name)
        current = os.stat(hidden, dir_fd=parent_fd, follow_symlinks=False)
        if not same_id(current, (root_dev, os.fstat(descriptor).st_ino)):
            raise RuntimeError("cleanup root changed during isolation")
        remove_tree(descriptor, root_dev, tree_plan)
        before = os.fstat(descriptor)
        current = os.stat(hidden, dir_fd=parent_fd, follow_symlinks=False)
        if stamp(current) != stamp(before):
            raise RuntimeError("cleanup root identity changed")
        latest = os.stat(hidden, dir_fd=parent_fd, follow_symlinks=False)
        if stamp(latest) != stamp(before):
            raise RuntimeError("cleanup root identity changed before removal")
        os.rmdir(hidden, dir_fd=parent_fd)
    finally:
        os.close(descriptor)


def rollback(parent_fd, tx_fd, candidate, output, candidate_fd, output_fd):
    try:
        if output_fd == -1:
            try:
                os.stat(output, dir_fd=parent_fd, follow_symlinks=False)
            except FileNotFoundError:
                return False
            retained = move_back(tx_fd, parent_fd, output, candidate, -1)
            if not retained:
                return False
            try:
                os.stat(output, dir_fd=parent_fd, follow_symlinks=False)
                return False
            except FileNotFoundError:
                return True
        current_cand = os.stat(candidate, dir_fd=tx_fd, follow_symlinks=False)
        current_output = os.stat(output, dir_fd=parent_fd, follow_symlinks=False)
        old_output = os.fstat(output_fd)
        if not same_id(current_cand, (old_output.st_dev, old_output.st_ino)):
            return False
        rename2(tx_fd, candidate, parent_fd, output, RENAME_EXCHANGE)
        os.fsync(tx_fd)
        os.fsync(parent_fd)
        restored = os.stat(output, dir_fd=parent_fd, follow_symlinks=False)
        private = os.stat(candidate, dir_fd=tx_fd, follow_symlinks=False)
        return same_id(restored, (old_output.st_dev, old_output.st_ino)) and same_id(
            private, (current_output.st_dev, current_output.st_ino)
        )
    except (OSError, RuntimeError):
        return False


def publish(parent, parent_id, candidate, candidate_seal, output, output_seal):
    check_name(candidate)
    check_name(output)
    expected_parent = parse_id(parent_id)
    parent_fd = os.open(parent, dir_flags())
    tx_name = ""
    tx_fd = -1
    candidate_fd = -1
    output_fd = -1
    published = False
    try:
        if not check_parent(parent, parent_fd, expected_parent):
            raise TxError(20, "publication parent identity changed")
        fcntl.flock(parent_fd, fcntl.LOCK_EX)
        if not check_parent(parent, parent_fd, expected_parent):
            raise TxError(20, "publication parent changed while locking")
        tx_name, tx_fd = make_tx(parent_fd, output)
        try:
            rename2(parent_fd, candidate, tx_fd, candidate, RENAME_NOREPLACE)
            os.fsync(tx_fd)
            os.fsync(parent_fd)
        except OSError as error:
            raise TxError(20, f"publication candidate isolation failed: {error}") from error
        candidate_fd = os.open(candidate, dir_flags(), dir_fd=tx_fd)
        candidate_info = os.fstat(candidate_fd)
        candidate_path = os.stat(candidate, dir_fd=tx_fd, follow_symlinks=False)
        if not same_id(candidate_path, (candidate_info.st_dev, candidate_info.st_ino)) or seal_fd(candidate_fd) != candidate_seal:
            raise TxError(20, "publication candidate changed before commit")
        try:
            output_fd = os.open(output, dir_flags(), dir_fd=parent_fd)
            output_info = os.fstat(output_fd)
            output_path = os.stat(output, dir_fd=parent_fd, follow_symlinks=False)
            if not same_id(output_path, (output_info.st_dev, output_info.st_ino)):
                raise TxError(20, "publication output identity changed before commit")
            current_output = seal_fd(output_fd)
        except FileNotFoundError:
            current_output = "-"
        if current_output != output_seal:
            raise TxError(20, "publication output changed before commit")
        candidate_path = os.stat(candidate, dir_fd=tx_fd, follow_symlinks=False)
        if not same_id(candidate_path, (candidate_info.st_dev, candidate_info.st_ino)) or seal_fd(candidate_fd) != candidate_seal:
            raise TxError(20, "publication candidate changed before rename")
        if output_fd != -1:
            output_path = os.stat(output, dir_fd=parent_fd, follow_symlinks=False)
            if not same_id(output_path, (output_info.st_dev, output_info.st_ino)) or seal_fd(output_fd) != output_seal:
                raise TxError(20, "publication output changed before rename")
        blocked = {signal.SIGHUP, signal.SIGINT, signal.SIGTERM}
        previous_mask = signal.pthread_sigmask(signal.SIG_BLOCK, blocked)
        try:
            abort_signal()
            flags = RENAME_NOREPLACE if output_seal == "-" else RENAME_EXCHANGE
            try:
                rename2(tx_fd, candidate, parent_fd, output, flags)
            except OSError as error:
                raise TxError(20, f"publication rename failed: {error}") from error
            published = True
        finally:
            signal.pthread_sigmask(signal.SIG_SETMASK, previous_mask)
        try:
            os.fsync(tx_fd)
            os.fsync(parent_fd)
        except OSError as error:
            raise TxError(21, f"publication parent sync failed: {error}") from error
        try:
            valid = check_parent(parent, parent_fd, expected_parent)
            current = os.stat(output, dir_fd=parent_fd, follow_symlinks=False)
            valid = valid and same_id(current, (candidate_info.st_dev, candidate_info.st_ino))
            valid = valid and seal_fd(candidate_fd) == candidate_seal
            if output_seal == "-":
                try:
                    os.stat(candidate, dir_fd=tx_fd, follow_symlinks=False)
                    valid = False
                except FileNotFoundError:
                    pass
            else:
                current = os.stat(candidate, dir_fd=tx_fd, follow_symlinks=False)
                valid = valid and same_id(current, (output_info.st_dev, output_info.st_ino))
                valid = valid and seal_fd(output_fd) == output_seal
        except (OSError, RuntimeError):
            valid = False
        if not valid:
            restored = rollback(parent_fd, tx_fd, candidate, output, candidate_fd, output_fd)
            if restored:
                published = False
            detail = " and the prior name was restored" if restored else " and both names were retained"
            raise TxError(22, "publication identities changed during commit" + detail)
        if output_seal != "-":
            try:
                remove_at(tx_fd, candidate, output_seal)
                os.fsync(tx_fd)
                os.fsync(parent_fd)
            except (OSError, RuntimeError) as error:
                raise TxError(23, f"published output cleanup failed: {error}") from error
        if not drop_tx(parent_fd, tx_name, tx_fd):
            raise TxError(23, "published output transaction directory is not empty")
        tx_name = ""
        return signal_code()
    except TxError as error:
        print(str(error), file=sys.stderr)
        return signal_code() or error.code
    finally:
        if tx_fd != -1 and not published:
            try:
                os.stat(candidate, dir_fd=tx_fd, follow_symlinks=False)
                restored_name = move_back(parent_fd, tx_fd, candidate, candidate, candidate_fd)
                if not restored_name:
                    print("publication candidate recovery was retained in the transaction directory", file=sys.stderr)
            except FileNotFoundError:
                pass
            except OSError as error:
                print(f"publication candidate recovery failed: {error}", file=sys.stderr)
        if tx_fd != -1 and tx_name:
            try:
                if drop_tx(parent_fd, tx_name, tx_fd):
                    tx_name = ""
            except OSError:
                pass
        if output_fd != -1:
            os.close(output_fd)
        if candidate_fd != -1:
            os.close(candidate_fd)
        if tx_fd != -1:
            os.close(tx_fd)
        os.close(parent_fd)


def discard(path, expected_id):
    parent = os.path.dirname(path)
    name = os.path.basename(path)
    check_name(name)
    expected = parse_id(expected_id)
    parent_fd = os.open(parent, dir_flags())
    tx_name = ""
    tx_fd = -1
    descriptor = -1
    deleting = False
    removed = False
    try:
        fcntl.flock(parent_fd, fcntl.LOCK_EX)
        tx_name, tx_fd = make_tx(parent_fd, name)
        rename2(parent_fd, name, tx_fd, name, RENAME_NOREPLACE)
        os.fsync(tx_fd)
        os.fsync(parent_fd)
        descriptor = os.open(name, dir_flags(), dir_fd=tx_fd)
        opened = os.fstat(descriptor)
        current = os.stat(name, dir_fd=tx_fd, follow_symlinks=False)
        if stamp(current) != stamp(opened) or not same_id(opened, expected):
            raise RuntimeError("discard directory identity changed")
        tree_seal = seal_fd(descriptor)
        deleting = True
        remove_at(tx_fd, name, tree_seal)
        removed = True
        if not drop_tx(parent_fd, tx_name, tx_fd):
            raise RuntimeError("discard transaction directory is not empty")
        tx_name = ""
    finally:
        if tx_fd != -1 and not deleting and not removed:
            try:
                os.stat(name, dir_fd=tx_fd, follow_symlinks=False)
                restored_name = move_back(parent_fd, tx_fd, name, name, descriptor)
                if not restored_name:
                    print("discard recovery was retained in the transaction directory", file=sys.stderr)
            except FileNotFoundError:
                pass
            except OSError as error:
                print(f"discard recovery failed: {error}", file=sys.stderr)
        if descriptor != -1:
            os.close(descriptor)
        if tx_fd != -1 and tx_name:
            try:
                if drop_tx(parent_fd, tx_name, tx_fd):
                    tx_name = ""
            except OSError:
                pass
        if tx_fd != -1:
            os.close(tx_fd)
        os.close(parent_fd)


def main():
    if len(sys.argv) == 3 and sys.argv[1] == "seal":
        seal_path(sys.argv[2])
        return signal_code()
    if len(sys.argv) == 4 and sys.argv[1] == "discard":
        discard(sys.argv[2], sys.argv[3])
        return signal_code()
    if len(sys.argv) == 8 and sys.argv[1] == "publish":
        return publish(*sys.argv[2:])
    print("invalid AIDL publication command", file=sys.stderr)
    return 2


if __name__ == "__main__":
    signal.signal(signal.SIGHUP, on_signal)
    signal.signal(signal.SIGINT, on_signal)
    signal.signal(signal.SIGTERM, on_signal)
    try:
        raise SystemExit(main())
    except (OSError, RuntimeError, ValueError) as error:
        print(str(error), file=sys.stderr)
        raise SystemExit(signal_code() or 2)
