package main

import (
	"os"

	"github.com/alecthomas/kingpin/v2"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	providerv1alpha1 "github.com/maximilianbraun/crossplane-provider-concourse/apis/v1alpha1"
	civ1alpha1 "github.com/maximilianbraun/crossplane-provider-concourse/apis/ci/v1alpha1"
	"github.com/maximilianbraun/crossplane-provider-concourse/internal/clients"
	"github.com/maximilianbraun/crossplane-provider-concourse/internal/controller"
)

func main() {
	var (
		app        = kingpin.New("provider-concourse", "Crossplane Concourse Provider").DefaultEnvars()
		debug      = app.Flag("debug", "Run with debug logging.").Short('d').Bool()
		syncPeriod = app.Flag("sync", "Controller sync period duration.").Default("1h").Duration()
	)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	zl := zap.New(zap.UseDevMode(*debug))
	ctrl.SetLogger(zl)
	log := ctrl.Log.WithName("provider-concourse")

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(providerv1alpha1.AddToScheme(scheme))
	utilruntime.Must(civ1alpha1.AddToScheme(scheme))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		log.Error(err, "unable to create manager")
		os.Exit(1)
	}

	_ = syncPeriod // TODO: wire into manager cache sync period

	cache := clients.NewCache()

	if err := controller.Setup(mgr, cache); err != nil {
		log.Error(err, "unable to setup controllers")
		os.Exit(1)
	}

	log.Info("starting provider")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "problem running manager")
		os.Exit(1)
	}
}
