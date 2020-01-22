package infrastructure

import (
	"context"
	"reflect"

	dodasv1alpha1 "github.com/dodas-ts/dodas-operator/pkg/apis/dodas/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_infrastructure")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

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
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &dodasv1alpha1.Infrastructure{},
	})
	if err != nil {
		return err
	}

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
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileInfrastructure) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Infrastructure")

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
		return reconcile.Result{}, err
	}

	dodasConfig := newDODASConfigForCR(instance)

	templateConfig := newConfigMapForCR(instance)

	// Define a new Pod object
	pod := newPodForCR(instance, templateConfig)

	// Set Infrastructure instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
		return reconcile.Result{}, err
	}
	if err := controllerutil.SetControllerReference(instance, templateConfig, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this template ConfigMap already exists
	foundTemplate := &corev1.ConfigMap{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: templateConfig.Name, Namespace: templateConfig.Namespace}, foundTemplate)
	if err != nil && errors.IsNotFound(err) {		

		reqLogger.Info("Creating a new ConfigMap", "ConfigMap.Namespace", templateConfig.Namespace, "Config.Name", templateConfig.Name)
		err = r.client.Create(context.TODO(), templateConfig)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// Check if this dodas ConfigMap already exists
	foundDodas := &corev1.ConfigMap{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: dodasConfig.Name, Namespace: dodasConfig.Namespace}, foundDodas)
	if err != nil && errors.IsNotFound(err) {		

		reqLogger.Info("Creating a new ConfigMap", "ConfigMap.Namespace", dodasConfig.Namespace, "Config.Name", dodasConfig.Name)
		err = r.client.Create(context.TODO(), dodasConfig)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// Check if this Pod already exists
	foundPod := &corev1.Pod{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, foundPod)
	if err != nil && errors.IsNotFound(err) {		

		reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
		err = r.client.Create(context.TODO(), pod)
		if err != nil {
			return reconcile.Result{}, err
		}
		
		instance.Status.InfID = "created"
		err = r.client.Status().Update(context.Background(), instance)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Pod created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	instance.Status.InfID = "Done"
	err = r.client.Status().Update(context.Background(), instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Pod already exists - don't requeue
	reqLogger.Info("Skip reconcile: Pod already exists", "Pod.Namespace", foundPod.Namespace, "Pod.Name", foundPod.Name)

	return reconcile.Result{}, nil
}


// newConfigMapForCR returns a configMap with the same name/namespace as the cr
func newDODASConfigForCR(cr *dodasv1alpha1.Infrastructure) *corev1.ConfigMap {

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-dodas",
			Namespace: cr.Namespace,
		},
		Data: map[string]string {
			"dodas.yml": cr.Spec.AuthFile,
			},
	}
}

// newConfigMapForCR returns a configMap with the same name/namespace as the cr
func newConfigMapForCR(cr *dodasv1alpha1.Infrastructure) *corev1.ConfigMap {

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-template",
			Namespace: cr.Namespace,
		},
		Data: map[string]string {
			"template.yml": cr.Spec.Template,
			},
	}
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newPodForCR(cr *dodasv1alpha1.Infrastructure, template *corev1.ConfigMap) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	var envs []corev1.EnvVar
    var env corev1.EnvVar

	fields := reflect.TypeOf(cr.Spec.CloudAuth)
	values := reflect.ValueOf(cr.Spec.CloudAuth)

	for i := 0; i < fields.NumField(); i++ {
		field := fields.Field(i)
		value := values.Field(i)
		if value.Interface() != "" {
			//keyTemp := fmt.Sprintf("%v = %v", decodeFields[field.Name], value.Interface())
			env = corev1.EnvVar{
				Name: "Cloud"+field.Name,
				Value: value.Interface().(string),
			}
			envs = append(envs, env)
		}
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			
			Volumes: []corev1.Volume {
				{
					Name: "template",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: cr.Name + "-template",
							},
				        },
					},
				},
				{
					Name: "dodas",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: cr.Name + "-dodas",
							},
				        },
					},
				},
			},
			Containers: []corev1.Container{
				{
					Name:    cr.Spec.Name,
					Image:   cr.Spec.Image,
					Env: envs,
					Args:[]string{
						"--config",
						"/etc/dodas.yml",
						"creat",
						"/etc/template.yml",
					},
					VolumeMounts:  []corev1.VolumeMount{
						{
							Name: "template",
							MountPath: "/etc/",
						},
						{
							Name: "dodas",
							MountPath: "/etc/",
						},
					},
				},
			},
		},
	}
}
