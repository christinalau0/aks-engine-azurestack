// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package transform

import (
	"os"
	"testing"

	"github.com/Jeffail/gabs"
	"github.com/onsi/gomega"
)

func TestAPIModelMergerMapValues(t *testing.T) {
	gomega.RegisterTestingT(t)

	m := make(map[string]APIModelValue)
	values := []string{
		"masterProfile.count=5",
		"agentPoolProfiles[0].name=agentpool1",
		"linuxProfile.adminUsername=admin",
		"servicePrincipalProfile.clientId='123a1238-c6eb-4b61-9d6f-7db6f1e14123',servicePrincipalProfile.secret='=!,Test$^='",
		"certificateProfile.etcdPeerCertificates[0]=certificate-value",
	}

	MapValues(m, values)
	gomega.Expect(m["masterProfile.count"].value).To(gomega.BeIdenticalTo(int64(5)))
	gomega.Expect(m["agentPoolProfiles[0].name"].arrayValue).To(gomega.BeTrue())
	gomega.Expect(m["agentPoolProfiles[0].name"].arrayIndex).To(gomega.BeIdenticalTo(0))
	gomega.Expect(m["agentPoolProfiles[0].name"].arrayProperty).To(gomega.BeIdenticalTo("name"))
	gomega.Expect(m["agentPoolProfiles[0].name"].arrayName).To(gomega.BeIdenticalTo("agentPoolProfiles"))
	gomega.Expect(m["agentPoolProfiles[0].name"].value).To(gomega.BeIdenticalTo("agentpool1"))
	gomega.Expect(m["linuxProfile.adminUsername"].value).To(gomega.BeIdenticalTo("admin"))
	gomega.Expect(m["servicePrincipalProfile.secret"].value).To(gomega.BeIdenticalTo("=!,Test$^="))
	gomega.Expect(m["servicePrincipalProfile.clientId"].value).To(gomega.BeIdenticalTo("123a1238-c6eb-4b61-9d6f-7db6f1e14123"))
	gomega.Expect(m["certificateProfile.etcdPeerCertificates[0]"].arrayValue).To(gomega.BeTrue())
	gomega.Expect(m["certificateProfile.etcdPeerCertificates[0]"].arrayIndex).To(gomega.BeIdenticalTo(0))
	gomega.Expect(m["certificateProfile.etcdPeerCertificates[0]"].arrayProperty).To(gomega.BeEmpty())
	gomega.Expect(m["certificateProfile.etcdPeerCertificates[0]"].arrayName).To(gomega.BeIdenticalTo("certificateProfile.etcdPeerCertificates"))
	gomega.Expect(m["certificateProfile.etcdPeerCertificates[0]"].value).To(gomega.BeIdenticalTo("certificate-value"))
}

func TestMergeValuesWithAPIModel(t *testing.T) {
	gomega.RegisterTestingT(t)

	m := make(map[string]APIModelValue)
	values := []string{
		"masterProfile.count=5",
		"agentPoolProfiles[0].name=agentpool1",
		"linuxProfile.adminUsername=admin",
		"certificateProfile.etcdPeerCertificates[0]=certificate-value",
	}

	MapValues(m, values)
	tmpFile, _ := MergeValuesWithAPIModel("../testdata/simple/kubernetes.json", m)

	jsonFileContent, err := os.ReadFile(tmpFile)
	gomega.Expect(err).To(gomega.BeNil())

	jsonAPIModel, err := gabs.ParseJSON(jsonFileContent)
	gomega.Expect(err).To(gomega.BeNil())

	masterProfileCount := jsonAPIModel.Path("properties.masterProfile.count").Data()
	gomega.Expect(masterProfileCount).To(gomega.BeIdenticalTo(float64(5)))

	adminUsername := jsonAPIModel.Path("properties.linuxProfile.adminUsername").Data()
	gomega.Expect(adminUsername).To(gomega.BeIdenticalTo("admin"))

	agentPoolProfileName := jsonAPIModel.Path("properties.agentPoolProfiles").Index(0).Path("name").Data().(string)
	gomega.Expect(agentPoolProfileName).To(gomega.BeIdenticalTo("agentpool1"))

	etcdPeerCertificates := jsonAPIModel.Path("properties.certificateProfile.etcdPeerCertificates").Index(0).Data()
	gomega.Expect(etcdPeerCertificates).To(gomega.BeIdenticalTo("certificate-value"))
}
