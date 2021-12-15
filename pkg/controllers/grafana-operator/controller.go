/*
Copyright 2021.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
		http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package grafana_operator

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	networkingv1 "k8s.io/api/networking/v1"

	"github.com/rhobs/monitoring-stack-operator/pkg/eventsource"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	integreatlyv1alpha1 "github.com/grafana-operator/grafana-operator/v4/api/integreatly/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	operatorsv1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
)

const (
	Namespace         = "monitoring-stack-operator"
	subscriptionName  = "monitoring-stack-operator-grafana-operator"
	operatorGroupName = "monitoring-stack-operator-grafana-operator"
	grafanaName       = "monitoring-stack-operator-grafana"
	grafanaCSV        = "grafana-operator.v4.1.0"
)

type reconciler struct {
	k8sClient        client.Client
	scheme           *runtime.Scheme
	logger           logr.Logger
	olmClientset     *versioned.Clientset
	k8sClientset     *kubernetes.Clientset
	grafanaClientset rest.Interface
}

type ReconcileFunc func(ctx context.Context) reconcileResult

type reconcileResult struct {
	ctrl.Result
	err  error
	stop bool
}

//+kubebuilder:rbac:groups="",resources=namespaces,verbs=list;watch,resourceNames=monitoring-stack-operator
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=create
//+kubebuilder:rbac:groups=operators.coreos.com,resources=subscriptions;operatorgroups,verbs=list;watch;create;update,namespace=monitoring-stack-operator
//+kubebuilder:rbac:groups=operators.coreos.com,resources=installplans,verbs=list;watch;update,namespace=monitoring-stack-operator
//+kubebuilder:rbac:groups=integreatly.org,namespace=monitoring-stack-operator,resources=grafanas,verbs=list;watch;create;update

// RegisterWithManager registers the controller with Manager
func RegisterWithManager(mgr ctrl.Manager) error {
	olmClientset, err := versioned.NewForConfig(mgr.GetConfig())
	if err != nil {
		return fmt.Errorf("could not create new OLM clientset: %w", err)
	}

	k8sClientset, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return fmt.Errorf("could not create new Kubernetes clientset: %w", err)
	}

	grafanaGVK := integreatlyv1alpha1.GroupVersion.WithKind("Grafana")
	grafanaClientset, err := apiutil.RESTClientForGVK(grafanaGVK, false, mgr.GetConfig(), serializer.NewCodecFactory(mgr.GetScheme()))
	if err != nil {
		return fmt.Errorf("could not create new Grafana clientset: %w", err)
	}

	log := ctrl.Log.WithName("grafana-operator")
	r := &reconciler{
		k8sClient:        mgr.GetClient(),
		scheme:           mgr.GetScheme(),
		logger:           log,
		olmClientset:     olmClientset,
		k8sClientset:     k8sClientset,
		grafanaClientset: grafanaClientset,
	}

	c, err := controller.New("grafana-operator", mgr, controller.Options{
		MaxConcurrentReconciles: 1,
		Reconciler:              r,
		Log:                     ctrl.Log,
	})
	if err != nil {
		return err
	}

	ticker := eventsource.NewTickerSource(30 * time.Minute)
	go ticker.Run()
	if err := c.Watch(ticker, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	namespaceInformer := r.namespaceInformer()
	go namespaceInformer.Run(wait.NeverStop)
	if err := c.Watch(&source.Informer{
		Informer: namespaceInformer,
	}, &handler.EnqueueRequestForObject{}, predicate.GenerationChangedPredicate{}); err != nil {
		return err
	}

	operatorGroupInformer := r.operatorGroupInformer()
	go operatorGroupInformer.Run(wait.NeverStop)
	if err := c.Watch(&source.Informer{
		Informer: operatorGroupInformer,
	}, &handler.EnqueueRequestForObject{}, predicate.GenerationChangedPredicate{}); err != nil {
		return err
	}

	subscriptionInformer := r.subscriptionInformer()
	go subscriptionInformer.Run(wait.NeverStop)
	if err := c.Watch(&source.Informer{
		Informer: subscriptionInformer,
	}, &handler.EnqueueRequestForObject{}, predicate.GenerationChangedPredicate{}); err != nil {
		return err
	}

	grafanaInformer := r.grafanaInformer()
	go grafanaInformer.Run(wait.NeverStop)
	if err := c.Watch(&source.Informer{
		Informer: grafanaInformer,
	}, &handler.EnqueueRequestForObject{}, predicate.GenerationChangedPredicate{}); err != nil {
		return err
	}

	installPlanInformer := r.installPlanInformer()
	go installPlanInformer.Run(wait.NeverStop)
	if err := c.Watch(&source.Informer{
		Informer: installPlanInformer,
	}, &handler.EnqueueRequestForObject{}, installPlanFilter{}); err != nil {
		return err
	}

	return nil
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.logger.WithValues("req", req)

	result := func(r reconcileResult) (ctrl.Result, error) {
		if r.err != nil {
			log.Error(r.err, "Error Reconciling resources")
		} else if r.Result.Requeue || r.Result.RequeueAfter != 0 {
			log.V(6).Info("Re-queueing after", "res", r.Result)
		} else {
			log.V(6).Info("Successfully reconciled resources")
		}
		return r.Result, r.err
	}

	log.V(6).Info("Reconciling resources")
	reconcilers := []ReconcileFunc{
		r.reconcileNamespace,
		r.reconcileOperatorGroup,
		r.reconcileSubscription,
		r.approveInstallPlan,
		r.reconcileGrafana,
	}
	for _, reconciler := range reconcilers {
		if res := reconciler(ctx); res.stop {
			return result(res)
		}
	}

	return result(reconcileResult{})
}

func (r *reconciler) reconcileNamespace(ctx context.Context) reconcileResult {
	log := r.logger.WithValues("Name", Namespace)
	log.V(6).Info("Reconciling namespace")

	key := types.NamespacedName{Name: Namespace}
	var namespace corev1.Namespace
	err := r.k8sClient.Get(ctx, key, &namespace)
	if err != nil && !errors.IsNotFound(err) {
		return reconcileError(err)
	}

	if errors.IsNotFound(err) {
		log.Info("Creating namespace")
		err = r.k8sClient.Create(ctx, NewNamespace())
		return creationResult(err)
	}

	// requeue if namespace is marked for deletion
	// TODO(sthaha): decide if want to use finalizers to prevent deletion but
	// we also need to solve how to properly cleanup / uninstall operator
	if namespace.Status.Phase != corev1.NamespaceActive {
		log.Info("Namespace is present but not active", "phase", namespace.Status.Phase)
		return end()
	}
	return next()
}

func (r *reconciler) reconcileOperatorGroup(ctx context.Context) reconcileResult {
	log := r.logger.WithValues("Name", operatorGroupName)
	log.V(6).Info("Reconciling OperatorGroup")

	key := types.NamespacedName{
		Name:      operatorGroupName,
		Namespace: Namespace,
	}
	var operatorGroup operatorsv1.OperatorGroup

	err := r.k8sClient.Get(ctx, key, &operatorGroup)
	if err != nil && !errors.IsNotFound(err) {
		return reconcileError(err)
	}

	// create
	desired := NewOperatorGroup()
	if errors.IsNotFound(err) {
		log.Info("Creating OperatorGroup")
		err := r.k8sClient.Create(ctx, desired)
		return creationResult(err)
	}

	// update
	if !reflect.DeepEqual(operatorGroup.Spec, desired.Spec) {
		log.Info("Updating OperatorGroup")
		operatorGroup.Spec = desired.Spec
		return updationResult(r.k8sClient.Update(ctx, &operatorGroup))
	}

	return next()
}

func (r *reconciler) reconcileSubscription(ctx context.Context) reconcileResult {
	log := r.logger.WithValues("Name", subscriptionName)
	key := types.NamespacedName{
		Name:      subscriptionName,
		Namespace: Namespace,
	}
	var subscription v1alpha1.Subscription
	err := r.k8sClient.Get(ctx, key, &subscription)
	if err != nil && !errors.IsNotFound(err) {
		return reconcileError(err)
	}

	// create
	desired := NewSubscription()
	if errors.IsNotFound(err) {
		log.Info("Creating Grafana Operator Subscription")
		err := r.k8sClient.Create(ctx, desired)
		return creationResult(err)
	}

	if subscription.Spec.StartingCSV == desired.Spec.StartingCSV {
		return next()
	}

	r.logger.WithValues("Name", subscription.Name).Info("Deleting Subscription")
	if err := r.k8sClient.Delete(ctx, &subscription); err != nil {
		return reconcileError(err)
	}

	r.logger.WithValues("Name", subscription.Status.InstalledCSV).Info("Deleting CSV")
	csv := v1alpha1.ClusterServiceVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "ClusterServiceVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      subscription.Status.InstalledCSV,
			Namespace: Namespace,
		},
	}
	if err := r.k8sClient.Delete(ctx, &csv); err != nil {
		return reconcileError(err)
	}

	r.logger.WithValues("Name", subscription.Name).Info("Creating Subscription")
	return creationResult(r.k8sClient.Create(ctx, &subscription))
}

func (r *reconciler) approveInstallPlan(ctx context.Context) reconcileResult {
	var installPlans v1alpha1.InstallPlanList
	err := r.k8sClient.List(ctx, &installPlans, client.InNamespace(Namespace))
	if err != nil {
		return reconcileError(err)
	}

	// wait for install plans to be created
	if len(installPlans.Items) == 0 {
		return end()
	}

	var approvePlan *v1alpha1.InstallPlan
	for _, installPlan := range installPlans.Items {
		csv := installPlan.Spec.ClusterServiceVersionNames[0]
		// ignore all but the install matching the Grafana version
		// also ignore install-plans that has an empty status
		if csv != grafanaCSV || len(installPlan.Status.BundleLookups) == 0 {
			continue
		}

		r.logger.V(6).Info("Found InstallPlan", "name", installPlan.Name, "csv", csv, "approved", installPlan.Spec.Approved)

		// look no further if an install plan for the desired CSV is already approved
		if installPlan.Spec.Approved {
			r.logger.V(6).Info("InstallPlan already approved", "name", installPlan.Name, "csv", csv)
			return next()
		}

		approvePlan = &installPlan
		break
	}

	// approvePlan can be nil if the install-plan for the desired version
	// hasn't been created or properly initialised yet
	if approvePlan == nil {
		return end()
	}

	r.logger.WithValues("Name", approvePlan.Name).Info("Approving InstallPlan")
	approvePlan.Spec.Approved = true
	return updationResult(r.k8sClient.Update(ctx, approvePlan))
}

func (r *reconciler) reconcileGrafana(ctx context.Context) reconcileResult {
	log := r.logger.WithValues("Name", grafanaName)
	key := types.NamespacedName{
		Name:      grafanaName,
		Namespace: Namespace,
	}

	var grafana integreatlyv1alpha1.Grafana
	err := r.k8sClient.Get(ctx, key, &grafana)
	if err != nil && !errors.IsNotFound(err) {
		// Ignore error and requeue if the errors are related to CRD not present
		if meta.IsNoMatchError(err) || errors.IsMethodNotSupported(err) {
			r.logger.V(6).Info("Grafana CRD does not exist - NoMatchError")
			return requeue(5*time.Second, nil)
		}

		return reconcileError(err)
	}

	// create
	desired := newGrafana()
	if errors.IsNotFound(err) {
		log.Info("Creating Grafana")
		err := r.k8sClient.Create(ctx, desired)

		// can fail because grafana operator hasn't created the CRD yet
		if errors.IsNotFound(err) || errors.IsMethodNotSupported(err) || meta.IsNoMatchError(err) {
			r.logger.V(6).Info("Grafana CRD is missing")
			return requeue(5*time.Second, nil)
		}

		return creationResult(err)
	}

	// update
	if !reflect.DeepEqual(desired.Spec, grafana.Spec) {
		log.Info("Updating Grafana")
		grafana.Spec = desired.Spec
		return updationResult(r.k8sClient.Update(ctx, &grafana))
	}

	return next()
}

func NewNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: Namespace,
		},
	}
}

func NewSubscription() *v1alpha1.Subscription {
	return &v1alpha1.Subscription{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "Subscription",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      subscriptionName,
			Namespace: Namespace,
		},
		Spec: &v1alpha1.SubscriptionSpec{
			CatalogSource:          "",
			CatalogSourceNamespace: "",
			Package:                "grafana-operator",
			Channel:                "v4",
			InstallPlanApproval:    v1alpha1.ApprovalManual,
			StartingCSV:            grafanaCSV,
			Config: &v1alpha1.SubscriptionConfig{
				Env: []corev1.EnvVar{
					{
						Name:  "DASHBOARD_NAMESPACES_ALL",
						Value: "true",
					},
				},
			},
		},
	}
}

func NewOperatorGroup() *operatorsv1.OperatorGroup {
	return &operatorsv1.OperatorGroup{
		TypeMeta: metav1.TypeMeta{
			APIVersion: operatorsv1.SchemeGroupVersion.String(),
			Kind:       "OperatorGroup",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      operatorGroupName,
			Namespace: Namespace,
		},
		Spec: operatorsv1.OperatorGroupSpec{
			TargetNamespaces: []string{
				Namespace,
			},
		},
	}
}

func newGrafana() *integreatlyv1alpha1.Grafana {
	flagTrue := true

	replicas := int32(1)
	maxUnavailable := intstr.FromInt(0)
	maxSurge := intstr.FromInt(1)
	return &integreatlyv1alpha1.Grafana{
		TypeMeta: metav1.TypeMeta{
			APIVersion: integreatlyv1alpha1.GroupVersion.String(),
			Kind:       "Grafana",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      grafanaName,
			Namespace: Namespace,
		},
		Spec: integreatlyv1alpha1.GrafanaSpec{
			Ingress: &integreatlyv1alpha1.GrafanaIngress{
				Enabled:  true,
				PathType: string(networkingv1.PathTypePrefix),
				Path:     "/",
			},
			Deployment: &integreatlyv1alpha1.GrafanaDeployment{
				Replicas: &replicas,
				Strategy: &v1.DeploymentStrategy{
					Type: v1.RollingUpdateDeploymentStrategyType,
					RollingUpdate: &v1.RollingUpdateDeployment{
						MaxUnavailable: &maxUnavailable,
						MaxSurge:       &maxSurge,
					},
				},
			},
			DashboardLabelSelector: []*metav1.LabelSelector{
				{
					MatchLabels: map[string]string{
						"app.kubernetes.io/part-of": "monitoring-stack-operator",
					},
				},
			},
			Config: integreatlyv1alpha1.GrafanaConfig{
				Log: &integreatlyv1alpha1.GrafanaConfigLog{
					Mode:  "console",
					Level: "info",
				},
				Auth: &integreatlyv1alpha1.GrafanaConfigAuth{
					DisableLoginForm:   &flagTrue,
					DisableSignoutMenu: &flagTrue,
				},
				AuthAnonymous: &integreatlyv1alpha1.GrafanaConfigAuthAnonymous{
					Enabled: &flagTrue,
				},
				Users: &integreatlyv1alpha1.GrafanaConfigUsers{
					ViewersCanEdit: &flagTrue,
				},
			},
		},
	}
}

func (r *reconciler) namespaceInformer() cache.SharedIndexInformer {
	clientset := r.k8sClientset.CoreV1().RESTClient()
	return singleResourceInformer(Namespace, "", "namespaces", &corev1.Namespace{}, clientset)
}

func (r *reconciler) subscriptionInformer() cache.SharedIndexInformer {
	clientset := r.olmClientset.OperatorsV1alpha1().RESTClient()
	return singleResourceInformer(subscriptionName, Namespace, "subscriptions", &v1alpha1.Subscription{}, clientset)
}

func (r *reconciler) operatorGroupInformer() cache.SharedIndexInformer {
	clientset := r.olmClientset.OperatorsV1().RESTClient()
	return singleResourceInformer(operatorGroupName, Namespace, "operatorgroups", &operatorsv1.OperatorGroup{}, clientset)
}

func (r *reconciler) grafanaInformer() cache.SharedIndexInformer {
	return singleResourceInformer(grafanaName, Namespace, "grafanas", &integreatlyv1alpha1.Grafana{}, r.grafanaClientset)
}

func (r *reconciler) installPlanInformer() cache.SharedIndexInformer {
	clientset := r.olmClientset.OperatorsV1alpha1().RESTClient()
	return singleNamespaceInformer(Namespace, "installplans", &v1alpha1.InstallPlan{}, clientset)
}

func singleResourceInformer(name string, namespace string, resource string, object runtime.Object, clientset rest.Interface) cache.SharedIndexInformer {
	listWatcher := cache.NewListWatchFromClient(
		clientset,
		resource,
		namespace,
		fields.AndSelectors(
			fields.OneTermEqualSelector("metadata.name", name),
		),
	)

	return cache.NewSharedIndexInformer(
		listWatcher,
		object,
		0,
		cache.Indexers{},
	)
}

func singleNamespaceInformer(namespace string, resource string, object runtime.Object, clientset rest.Interface) cache.SharedIndexInformer {
	listWatcher := cache.NewListWatchFromClient(
		clientset,
		resource,
		namespace,
		fields.AndSelectors(
			fields.OneTermEqualSelector("metadata.namespace", namespace),
		),
	)

	return cache.NewSharedIndexInformer(
		listWatcher,
		object,
		0,
		cache.Indexers{},
	)
}

func creationResult(err error) reconcileResult {

	// requeue on creation
	if err == nil {
		return end()
	}

	// do not requeue if object exists
	if errors.IsAlreadyExists(err) {
		return next()
	}

	return reconcileError(err)
}

// returns whether to requeue
func updationResult(err error) reconcileResult {
	// do not requeue if updation is successful since the informer should
	// trigger a reconcilation loop
	if err == nil {
		return next()
	}

	// requeue if the cache is invalid and do not log error
	if errors.IsConflict(err) {
		return requeue(2*time.Second, nil)
	}

	return reconcileError(err)
}

// end returns a reconcile result that terminates the current loop
// and doesn't requeue
func end() reconcileResult {
	return reconcileResult{
		stop: true,
	}
}

func next() reconcileResult {
	return reconcileResult{
		stop: false,
	}
}

func requeue(d time.Duration, err error) reconcileResult {

	res := ctrl.Result{RequeueAfter: d}
	if d == 0 {
		res.Requeue = true
	}
	return reconcileResult{
		stop:   true,
		err:    err,
		Result: res,
	}
}

func reconcileError(err error) reconcileResult {
	return reconcileResult{
		stop: true,
		err:  err,
	}
}
