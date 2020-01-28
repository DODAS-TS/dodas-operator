package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"time"

	dodasv1alpha1 "github.com/dodas-ts/dodas-operator/pkg/apis/dodas/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	//"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_infrastructure")

// Add creates a new Infrastructure Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileInfrastructure{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("infrastructure-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Infrastructure
	err = c.Watch(&source.Kind{Type: &dodasv1alpha1.Infrastructure{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Infrastructure
	// err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
	// 	IsController: true,
	// 	OwnerType:    &dodasv1alpha1.Infrastructure{},
	// })
	// if err != nil {
	// 	return err
	// }

	return nil
}

// blank assignment to verify that ReconcileInfrastructure implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileInfrastructure{}

// ReconcileInfrastructure reconciles a Infrastructure object
type ReconcileInfrastructure struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Infrastructure object and makes changes based on the state read
// and what is in the Infrastructure.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileInfrastructure) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Infrastructure")

	delay := time.Minute
	// Fetch the Infrastructure instance
	instance := &dodasv1alpha1.Infrastructure{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Info(err.Error())
		return reconcile.Result{RequeueAfter: delay}, nil
	}

	// TODO: if status == precendente
	// What happens if edit when already defined?
	// TODO: check for deletionTimestamp set
	// TODO: if infID but updated

	// everything ok go on
	if (instance.Status.InfID != "") && (instance.Status.Error == "") {

		if instance.GetDeletionTimestamp() == nil {
			// TODO: check if present in IM
			return reconcile.Result{}, nil
		}

		// TODO: create function delete
		// TODO: refresh token and delete

		reqLogger.Info("Destroying cluster before deleting resource")

		err := DeleteInf(instance)
		if err != nil {
			reqLogger.Error(err, "Failed to remove Inf")
			reqLogger.Info(fmt.Sprintf("Reconciling %s in %s", instance.Name, delay))
			return reconcile.Result{RequeueAfter: delay}, nil
		}
		reqLogger.Info(fmt.Sprintf("Removed infrastracture ID: %s", instance.Status.InfID))

		// remove finalizer
		reqLogger.Info("Deletion successful, removing finalizer")
		instance.SetFinalizers([]string{})
		err = r.client.Update(context.Background(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Infrastructure finalizer")
			return reconcile.Result{RequeueAfter: delay}, nil
		}
		return reconcile.Result{RequeueAfter: delay}, nil
	}

	// Check if requested template ConfigMap already exists
	foundTemplate := &corev1.ConfigMap{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.Template, Namespace: instance.Namespace}, foundTemplate)
	if err != nil && errors.IsNotFound(err) {
		// if error persists retry later
		errorMsg := fmt.Sprintf("No template found with name: %s", instance.Spec.Template)
		reqLogger.Error(err, errorMsg)
		if instance.Status.Error != errorMsg {
			instance.Status.Error = errorMsg
			instance.Status.Status = "error"
			r.client.Status().Update(context.Background(), instance)
		}
		reqLogger.Info(fmt.Sprintf("Reconciling %s in %s", instance.Name, delay))
		return reconcile.Result{RequeueAfter: delay}, nil
	} else if err != nil {
		errorMsg := "Error looking for template"
		reqLogger.Error(err, errorMsg)
		if instance.Status.Error != errorMsg {
			instance.Status.Error = errorMsg
			instance.Status.Status = "error"
			r.client.Status().Update(context.Background(), instance)
		}
		reqLogger.Info(fmt.Sprintf("Reconciling %s in %s", instance.Name, delay))
		return reconcile.Result{RequeueAfter: delay}, nil
	}

	var templateContent map[string]string
	//map[string][]byte
	// check if template file is there
	templateContent = foundTemplate.Data

	for _, value := range templateContent {
		templateBytes := value

		instance.Status.Status = "creating"
		r.client.Status().Update(context.Background(), instance)

		// Insert finalizer before starting the creation
		instance.SetFinalizers([]string{"delete"})
		err = r.client.Update(context.Background(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update finalizer")
			return reconcile.Result{RequeueAfter: delay}, nil
		}

		infID, err := CreateInf(instance, []byte(templateBytes))
		if err != nil {
			reqLogger.Error(err, "Failed to create Inf")
			if instance.Status.Error != err.Error() {
				instance.Status.Error = err.Error()
				instance.Status.Status = "error"
				r.client.Status().Update(context.Background(), instance)
			}
			reqLogger.Info(fmt.Sprintf("Reconciling %s in %s", instance.Name, delay))
			return reconcile.Result{RequeueAfter: delay}, nil
		}

		instance.Status.InfID = infID
		instance.Status.Error = ""
		instance.Status.Status = "created"
		r.client.Status().Update(context.Background(), instance)
		break
	}
	// GET TOKEN and SAVE refresh

	return reconcile.Result{}, nil
}

var decodeFields = map[string]string{
	"ID":            "id",
	"Type":          "type",
	"Username":      "username",
	"Password":      "password",
	"Token":         "token",
	"Host":          "host",
	"Tenant":        "tenant",
	"AuthURL":       "auth_url",
	"AuthVersion":   "auth_version",
	"Domain":        "domain",
	"ServiceRegion": "service_region",
}

// PrepareAuthHeaders ..
func PrepareAuthHeaders(clientConf *dodasv1alpha1.Infrastructure) string {

	var authHeaderCloudList []string

	fields := reflect.TypeOf(clientConf.Spec.CloudAuth)
	values := reflect.ValueOf(clientConf.Spec.CloudAuth)

	// TODO: use go templates!

	for i := 0; i < fields.NumField(); i++ {
		field := fields.Field(i)
		value := values.Field(i)

		if value.Interface() != "" {
			keyTemp := fmt.Sprintf("%v = %v", decodeFields[field.Name], value)
			authHeaderCloudList = append(authHeaderCloudList, keyTemp)
		}
	}

	authHeaderCloud := strings.Join(authHeaderCloudList, ";")

	var authHeaderIMList []string

	fields = reflect.TypeOf(clientConf.Spec.ImAuth)
	values = reflect.ValueOf(clientConf.Spec.ImAuth)

	for i := 0; i < fields.NumField(); i++ {
		field := fields.Field(i)
		if decodeFields[field.Name] != "host" {
			value := values.Field(i)
			if value.Interface() != "" {
				keyTemp := fmt.Sprintf("%v = %v", decodeFields[field.Name], value.Interface())
				authHeaderIMList = append(authHeaderIMList, keyTemp)
			}
		}
	}

	authHeaderIM := strings.Join(authHeaderIMList, ";")

	authHeader := authHeaderCloud + "\\n" + authHeaderIM

	//fmt.Printf(authHeader)

	return authHeader
}

// RefreshToken ..
func RefreshToken(refreshToken string, clientConf *dodasv1alpha1.Infrastructure) (string, error) {

	var token string

	clientID := clientConf.Spec.AllowRefresh.ClientID
	clientSecret := clientConf.Spec.AllowRefresh.ClientSecret
	IAMTokenEndpoint := clientConf.Spec.AllowRefresh.IAMTokenEndpoint

	client := &http.Client{
		Timeout: 300 * time.Second,
	}

	req, _ := http.NewRequest("GET", IAMTokenEndpoint, nil)

	req.SetBasicAuth(clientID, clientSecret)

	req.Header.Set("grant_type", "refresh_token")
	req.Header.Set("refresh_token", refreshToken)

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode == 200 {

		type accessTokenStruct struct {
			AccessToken string `json:"access_token"`
		}

		var accessTokenJSON accessTokenStruct

		err = json.Unmarshal(body, &accessTokenJSON)
		if err != nil {
			return "", err
		}

		token = accessTokenJSON.AccessToken

	} else {
		return "", fmt.Errorf("ERROR: %s", string(body))
	}

	return token, nil
}

// newConfigMapForCR returns a configMap with the same name/namespace as the cr
// func newConfigMapForCR(cr *dodasv1alpha1.Infrastructure) *corev1.ConfigMap {

// 	return &corev1.ConfigMap{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      cr.Name + "-template",
// 			Namespace: cr.Namespace,
// 		},
// 		Data: map[string]string {
// 			"template.yml": cr.Spec.Template,
// 			"dodas.yml": cr.Spec.AuthFile,
// 			"inf.id": "",
// 			},
// 	}
// }
