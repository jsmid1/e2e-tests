package has

import (
	"context"
	"fmt"
	"time"

	appservice "github.com/redhat-appstudio/application-api/api/v1alpha1"
	"github.com/redhat-appstudio/e2e-tests/pkg/utils"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	rclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Contains all methods related with Applications objects CRUD operations.
type ApplicationsInterface interface {
	// Returns an application obj from the kubernetes cluster.
	GetApplication(name string, namespace string) (*appservice.Application, error)

	// Given a devfile content determine if a gitops repository was created in GitHub.
	ApplicationGitopsRepoExists(devfileContent string) wait.ConditionFunc

	// Creates an application object in the kubernetes cluster.
	CreateApplication(name string, namespace string) (*appservice.Application, error)

	// Creates an application object in the kubernetes cluster and wait for a period of given timeout.
	CreateApplicationWithTimeout(name string, namespace string, timeout time.Duration) (*appservice.Application, error)

	// Deletes an application object from the kubernetes cluster.
	DeleteApplication(name string, namespace string, reportErrorOnNotFound bool) error

	// Deletes all applications from the given namespace in the kubernetes cluster.
	DeleteAllApplicationsInASpecificNamespace(namespace string, timeout time.Duration) error
}

// GetApplication returns an application given a name and namespace from kubernetes cluster.
func (h *hasFactory) GetApplication(name string, namespace string) (*appservice.Application, error) {
	application := appservice.Application{
		Spec: appservice.ApplicationSpec{},
	}
	if err := h.KubeRest().Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &application); err != nil {
		return nil, err
	}

	return &application, nil
}

// ApplicationDevfilePresent check if devfile exists in the application status.
func (h *hasFactory) ApplicationDevfilePresent(application *appservice.Application) wait.ConditionFunc {
	return func() (bool, error) {
		app, err := h.GetApplication(application.Name, application.Namespace)
		if err != nil {
			return false, nil
		}
		application.Status = app.Status
		return application.Status.Devfile != "", nil
	}
}

// ApplicationGitopsRepoExists check from the devfile content if application-service creates a gitops repo in GitHub.
func (s *hasFactory) ApplicationGitopsRepoExists(devfileContent string) wait.ConditionFunc {
	return func() (bool, error) {
		gitOpsRepoURL := utils.ObtainGitOpsRepositoryName(devfileContent)
		return s.Github.CheckIfRepositoryExist(gitOpsRepoURL), nil
	}
}

// CreateApplication creates an application in the kubernetes cluster with 10 minutes default time for creation.
func (h *hasFactory) CreateApplication(name string, namespace string) (*appservice.Application, error) {
	return h.CreateApplicationWithTimeout(name, namespace, time.Minute*10)
}

// CreateHasApplicationWithTimeout creates an application in the kubernetes cluster with a custom default time for creation.
func (h *hasFactory) CreateApplicationWithTimeout(name string, namespace string, timeout time.Duration) (*appservice.Application, error) {
	application := &appservice.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appservice.ApplicationSpec{
			DisplayName: name,
		},
	}

	if err := h.KubeRest().Create(context.TODO(), application); err != nil {
		return nil, err
	}

	if err := utils.WaitUntil(h.ApplicationDevfilePresent(application), timeout); err != nil {
		application = h.refreshApplicationForErrorDebug(application)
		return nil, fmt.Errorf("timed out when waiting for devfile content creation for application %s in %s namespace: %+v. applicattion: %s", name, namespace, err, utils.ToPrettyJSONString(application))
	}

	return application, nil
}

// DeleteApplication delete a HAS Application resource from the namespace.
// Optionally, it can avoid returning an error if the resource did not exist:
// - specify 'false', if it's likely the Application has already been deleted (for example, because the Namespace was deleted)
func (h *hasFactory) DeleteApplication(name string, namespace string, reportErrorOnNotFound bool) error {
	application := appservice.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	if err := h.KubeRest().Delete(context.TODO(), &application); err != nil {
		if !k8sErrors.IsNotFound(err) || (k8sErrors.IsNotFound(err) && reportErrorOnNotFound) {
			return fmt.Errorf("error deleting an application: %+v", err)
		}
	}
	return utils.WaitUntil(h.ApplicationDeleted(&application), 1*time.Minute)
}

// ApplicationDeleted check if a given application object was deleted successfully from the kubernetes cluster.
func (h *hasFactory) ApplicationDeleted(application *appservice.Application) wait.ConditionFunc {
	return func() (bool, error) {
		_, err := h.GetApplication(application.Name, application.Namespace)
		return err != nil && k8sErrors.IsNotFound(err), nil
	}
}

// DeleteAllApplicationsInASpecificNamespace removes all application CRs from a specific namespace. Useful when creating a lot of resources and want to remove all of them
func (h *hasFactory) DeleteAllApplicationsInASpecificNamespace(namespace string, timeout time.Duration) error {
	if err := h.KubeRest().DeleteAllOf(context.TODO(), &appservice.Application{}, rclient.InNamespace(namespace)); err != nil {
		return fmt.Errorf("error deleting applications from the namespace %s: %+v", namespace, err)
	}

	applicationList := &appservice.ApplicationList{}
	return utils.WaitUntil(func() (done bool, err error) {
		if err := h.KubeRest().List(context.Background(), applicationList, &rclient.ListOptions{Namespace: namespace}); err != nil {
			return false, nil
		}
		return len(applicationList.Items) == 0, nil
	}, timeout)
}

// refreshApplicationForErrorDebug return the latest application object from the kubernetes cluster.
func (h *hasFactory) refreshApplicationForErrorDebug(application *appservice.Application) *appservice.Application {
	retApp := &appservice.Application{}

	if err := h.KubeRest().Get(context.Background(), rclient.ObjectKeyFromObject(application), retApp); err != nil {
		return application
	}

	return retApp
}
