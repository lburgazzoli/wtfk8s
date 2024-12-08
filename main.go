package main

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/restmapper"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/sirupsen/logrus"
)

var (
	kubeConfig string
	namespace  string
)

func run(cmd *cobra.Command, args []string) error {
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		return err
	}

	cli, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return err
	}

	disc := discovery.NewDiscoveryClientForConfigOrDie(cfg)
	groupResources, err := restmapper.GetAPIGroupResources(disc)
	if err != nil {
		return err
	}

	gvr := schema.GroupVersionResource{}

	switch strings.Count(args[0], ".") {
	case 0:
		gvr.Resource = args[0]
	case 1:
		s := strings.SplitN(args[0], ".", 2)
		gvr.Resource = s[0]
		gvr.Group = s[1]
	default:
		return fmt.Errorf("unknown group version %q", args[0])
	}

	for _, groupResource := range groupResources {
		//
	}

	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(cli, time.Minute, namespace, nil)
	informer := factory.ForResource(gvr).Informer()

	_, err = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			fmt.Printf("%s\n", cmp.Diff(oldObj, newObj))
		},
	})

	if err != nil {
		return err
	}

	<-cmd.Context().Done()

	return nil

}

func main() {
	// klog.SetOutput(io.)
	logrus.SetLevel(logrus.InfoLevel)

	cmd := &cobra.Command{
		Use:          "wtfk8s",
		Short:        "wtfk8s",
		SilenceUsage: true,
		RunE:         run,
	}

	cmd.Flags().StringVar(&kubeConfig, "kubeconfig", os.Getenv("KUBECONFIG"), "")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", corev1.NamespaceAll, "")

	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
