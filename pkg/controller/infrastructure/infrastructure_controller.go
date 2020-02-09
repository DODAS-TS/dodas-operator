package infrastructure

import (
	"context"
	"fmt"
	"regexp"
	"time"

	dodas "github.com/dodas-ts/dodas-go-client/pkg/utils"
	dodasv1alpha1 "github.com/dodas-ts/dodas-operator/pkg/apis/dodas/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

	dodasClient := dodas.Conf{
		Im:           dodas.ConfIM(instance.Spec.ImAuth),
		Cloud:        dodas.ConfCloud(instance.Spec.CloudAuth),
		AllowRefresh: dodas.TokenRefreshConf(instance.Spec.AllowRefresh),
	}

	// Define a new secret  object for token and refresh token
	refreshSecret := newRefreshToken(instance)

	// Set instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, refreshSecret, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	if instance.Spec.AllowRefresh.IAMTokenEndpoint != "" {
		// Check if this Secret for tokens already exists
		found := &corev1.Secret{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: refreshSecret.Name, Namespace: refreshSecret.Namespace}, found)
		if err != nil && errors.IsNotFound(err) {
			reqLogger.Info("Creating a new Pod", "refreshSecret.Namespace", refreshSecret.Namespace, "refreshSecret.Name", refreshSecret.Name)
			err = r.client.Create(context.TODO(), refreshSecret)
			if err != nil {
				return reconcile.Result{RequeueAfter: delay}, err
			}
		} else if err != nil {
			return reconcile.Result{RequeueAfter: delay}, err
		}

		// Get refresh token if not set
		if refreshSecret.StringData["RefreshToken"] == "" {

			refreshToken, err := dodasClient.GetRefreshToken()
			if err != nil {
				return reconcile.Result{}, err
			}

			reqLogger.Info(fmt.Sprintf("Adding refresh token to secret: %s", refreshToken))
			refreshSecret.StringData["RefreshToken"] = refreshToken
			err = r.client.Update(context.Background(), refreshSecret)
			if err != nil {
				return reconcile.Result{}, err
			}
			reqLogger.Info(fmt.Sprintf("Adding refresh token added: %s", refreshToken))
		}

		var accessToken string

		// Check if access token is valid otherwise
		_, err = dodasClient.ListInfIDs()
		if err != nil {
			re := regexp.MustCompile(`^.*OIDC auth Token expired`)
			if re.Match([]byte(err.Error())) {
				reqLogger.Info(fmt.Sprintf("Token expired. Using refresh token to get the access one: %s", refreshSecret.StringData["RefreshToken"]))

				accessToken, err = dodasClient.GetAccessToken(refreshSecret.StringData["RefreshToken"])
				if err != nil {
					panic(err)
				}

				refreshSecret.StringData["AccessToken"] = accessToken
				err = r.client.Update(context.Background(), refreshSecret)
				if err != nil {
					return reconcile.Result{}, err
				}

			}
		} else {
			accessToken = refreshSecret.StringData["AccessToken"]
		}

		// TODO: Check wheter cloud or im uses the token auth
		instance.Spec.ImAuth.Token = accessToken
		instance.Spec.CloudAuth.Password = accessToken

	}

	// What happens if edit when already defined?
	// TODO: use dodas client update to check if the template is changed
	// TODO: check for deletionTimestamp set
	// TODO: update status and outputs

	// if creation failed somehow, allow deletion
	if (instance.Status.InfID == "") && (instance.Status.Status == "creating") {
		// remove finalizer
		reqLogger.Info("Creation failed somehow, allow deletion")
		instance.SetFinalizers([]string{})
		err = r.client.Update(context.Background(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Infrastructure finalizer")
			return reconcile.Result{RequeueAfter: delay}, nil
		}
		return reconcile.Result{RequeueAfter: delay}, nil
	}

	// everything ok go on
	if (instance.Status.InfID != "") && (instance.Status.Error == "") {

		if instance.GetDeletionTimestamp() == nil {
			// TODO: check if present in IM
			return reconcile.Result{}, nil
		}

		// TODO: create function delete
		// TODO: refresh token and delete

		reqLogger.Info("Destroying cluster before deleting resource")

		err := dodasClient.DestroyInf(instance.Status.InfID)
		// TODO: check if error is: infra do not exist anymore
		if err != nil {
			reqLogger.Error(err, "Failed to remove Inf")
			reqLogger.Info(fmt.Sprintf("Reconciling %s in %s", instance.Name, delay))

			instance.Status.Error = fmt.Sprintf("Failed to destro Infra: %s", err.Error())
			instance.Status.Status = "error"
			r.client.Status().Update(context.Background(), instance)

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

		// TODO: pass bytes instead of file
		err := dodasClient.Validate([]byte(templateBytes))
		if err != nil {
			reqLogger.Error(err, "Invalid template")
			if instance.Status.Error != err.Error() {
				instance.Status.Error = err.Error()
				instance.Status.Status = "error"
				r.client.Status().Update(context.Background(), instance)
			}
		}

		infID, err := dodasClient.CreateInf([]byte(templateBytes))
		if err != nil {
			reqLogger.Error(err, "Failed to create Inf")

			// TODO: remove finalizer
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

func newRefreshToken(cr *dodasv1alpha1.Infrastructure) *corev1.Secret {

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-token",
			Namespace: cr.Namespace,
		},
		StringData: map[string]string{
			"RefreshToken": "",
			"AccessToken":  cr.Spec.ImAuth.Token,
		},
	}
}
