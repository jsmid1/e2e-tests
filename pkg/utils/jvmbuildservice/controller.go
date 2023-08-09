package jvmbuildservice

import (
	kubeCl "github.com/redhat-appstudio/e2e-tests/pkg/apis/kubernetes"
)

type JvmbuildserviceController struct {
	*kubeCl.CustomClient
}

func NewSuiteControler(kube *kubeCl.CustomClient) (*JvmbuildserviceController, error) {
	return &JvmbuildserviceController{
		kube,
	}, nil
}
