// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package armhelpers

import (
	"context"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
)

const (
	badlocation  = "badlocation"
	badPublisher = "badPublisher"
	badOffer     = "badOffer"
	badSku       = "badSku"
)

func TestVMImageFetcherInterface(t *testing.T) {
	mc, err := NewHTTPMockClient()
	if err != nil {
		t.Fatalf("failed to create HttpMockClient - %s", err)
	}
	mc.RegisterLogin()
	mc.RegisterVMImageFetcherInterface()

	err = mc.Activate()
	if err != nil {
		t.Fatalf("failed to activate HttpMockClient - %s", err)
	}
	defer mc.DeactivateAndReset()

	options := &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			InsecureAllowCredentialWithHTTP: true,
			Cloud:                           mc.GetEnvironment(),
		},
	}
	azureClient, err := NewAzureClient(subscriptionID, &fake.TokenCredential{}, options)
	if err != nil {
		t.Fatalf("can not get client %s", err)
	}

	_, err = azureClient.GetVirtualMachineImage(context.Background(), location, publisher, offer, sku, version)
	if err != nil {
		t.Error(err)
	}

	_, err = azureClient.ListVirtualMachineImages(context.Background(), location, publisher, offer, sku)
	if err != nil {
		t.Error(err)
	}
}

func TestVMImageFetcherInterfaceBadInput(t *testing.T) {
	mc, err := NewHTTPMockClient()
	if err != nil {
		t.Fatalf("failed to create HttpMockClient - %s", err)
	}
	mc.RegisterLogin()
	mc.RegisterVMImageFetcherInterface()

	err = mc.Activate()
	if err != nil {
		t.Fatalf("failed to activate HttpMockClient - %s", err)
	}
	defer mc.DeactivateAndReset()

	options := &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			InsecureAllowCredentialWithHTTP: true,
			Cloud:                           mc.GetEnvironment(),
		},
	}
	azureClient, err := NewAzureClient(subscriptionID, &fake.TokenCredential{}, options)
	if err != nil {
		t.Fatalf("can not get client %s", err)
	}

	_, err = azureClient.GetVirtualMachineImage(context.Background(), badlocation, badPublisher, badOffer, badSku, version)
	if err == nil {
		t.Fatal("GetVirtualMachineImage did not fail with bad input")
	}

	_, err = azureClient.ListVirtualMachineImages(context.Background(), badlocation, badPublisher, badOffer, badSku)
	if err == nil {
		t.Fatal("ListVirtualMachineImages did not fail with bad input")
	}
}
