// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"errors"
	"fmt"
	"strings"
)

type pkgDef struct {
	Path    string
	Kind    string
	Version string
	Api     string
	Ext     string
	Rev     string
	Display string
	LicID   string
	Compact bool
	Deps    bool
}

const xmlHeader = "<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"yes\"?>"
const xmlRepo = "<ns2:repository xmlns:ns2=\"http://schemas.android.com/repository/android/common/02\" xmlns:sdk=\"http://schemas.android.com/sdk/android/repo/repository2/03\" xmlns:generic=\"http://schemas.android.com/repository/android/generic/02\" xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\">"
const compactRepo = "<ns2:repository xmlns:ns2=\"http://schemas.android.com/repository/android/common/02\" xmlns:ns3=\"http://schemas.android.com/repository/android/common/01\" xmlns:ns4=\"http://schemas.android.com/repository/android/generic/01\" xmlns:ns5=\"http://schemas.android.com/repository/android/generic/02\" xmlns:ns6=\"http://schemas.android.com/sdk/android/repo/addon2/01\" xmlns:ns7=\"http://schemas.android.com/sdk/android/repo/addon2/02\" xmlns:ns8=\"http://schemas.android.com/sdk/android/repo/addon2/03\" xmlns:ns9=\"http://schemas.android.com/sdk/android/repo/repository2/01\" xmlns:ns10=\"http://schemas.android.com/sdk/android/repo/repository2/02\" xmlns:ns11=\"http://schemas.android.com/sdk/android/repo/repository2/03\" xmlns:ns12=\"http://schemas.android.com/sdk/android/repo/sys-img2/04\" xmlns:ns13=\"http://schemas.android.com/sdk/android/repo/sys-img2/03\" xmlns:ns14=\"http://schemas.android.com/sdk/android/repo/sys-img2/02\" xmlns:ns15=\"http://schemas.android.com/sdk/android/repo/sys-img2/01\">"

func packageXML(pkg pkgDef, license string) (string, error) {
	var body strings.Builder
	if pkg.Compact {
		body.WriteString(xmlHeader + compactRepo)
		fmt.Fprintf(&body, "<license id=\"%s\" type=\"text\"/><localPackage path=\"%s\" obsolete=\"false\">", pkg.LicID, pkg.Path)
		if pkg.Kind == "platform" {
			fmt.Fprintf(&body, "<type-details xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xsi:type=\"ns11:platformDetailsType\"><api-level>%s</api-level><extension-level>%s</extension-level><base-extension>true</base-extension><layoutlib api=\"15\"/></type-details><revision><major>%s</major></revision>", pkg.Api, pkg.Ext, pkg.Rev)
		} else {
			body.WriteString("<type-details xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xsi:type=\"ns5:genericDetailsType\"/><revision>")
			for index, value := range strings.Split(pkg.Version, ".") {
				name := []string{"major", "minor", "micro"}[index]
				fmt.Fprintf(&body, "<%s>%s</%s>", name, value, name)
			}
			body.WriteString("</revision>")
		}
		fmt.Fprintf(&body, "<display-name>%s</display-name><uses-license ref=\"%s\"/>", pkg.Display, pkg.LicID)
		if pkg.Deps {
			body.WriteString("<dependencies><dependency path=\"tools\"><min-revision><major>22</major></min-revision></dependency></dependencies>")
		}
		body.WriteString("</localPackage></ns2:repository>")
		return body.String(), nil
	}
	if license == "" {
		return "", errors.New("生成包元数据缺少固定SDK许可输入")
	}
	body.WriteString(xmlHeader + "\n" + xmlRepo + "\n" + license)
	fmt.Fprintf(&body, "  <localPackage path=\"%s\" obsolete=\"false\">\n", pkg.Path)
	if pkg.Kind == "platform" {
		fmt.Fprintf(&body, "    <type-details xsi:type=\"sdk:platformDetailsType\">\n      <api-level>%s</api-level>\n      <codename></codename>\n      <extension-level>%s</extension-level>\n      <base-extension>true</base-extension>\n      <layoutlib api=\"15\"/>\n    </type-details>\n    <revision>\n      <major>%s</major>\n    </revision>\n", pkg.Api, pkg.Ext, pkg.Rev)
	} else {
		body.WriteString("    <type-details xsi:type=\"generic:genericDetailsType\"/>\n    <revision>\n")
		for index, value := range strings.Split(pkg.Version, ".") {
			name := []string{"major", "minor", "micro"}[index]
			fmt.Fprintf(&body, "      <%s>%s</%s>\n", name, value, name)
		}
		body.WriteString("    </revision>\n")
	}
	fmt.Fprintf(&body, "    <display-name>%s</display-name>\n    <uses-license ref=\"android-sdk-license\"/>\n  </localPackage>\n</ns2:repository>\n", pkg.Display)
	return body.String(), nil
}
