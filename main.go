package main

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	kubeConfig    string
	namespace     string
	labelSelector string
	fieldSelector string
	gvr           schema.GroupVersionResource
	managedFields bool
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

	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(cli, time.Minute, namespace, func(opts *metav1.ListOptions) {
		opts.LabelSelector = labelSelector
		opts.FieldSelector = fieldSelector
	})

	informer := factory.ForResource(gvr).Informer()

	_, err = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if in, ok := obj.(client.Object); ok {
				fmt.Println("add", "gvk", in.GetObjectKind().GroupVersionKind(), "ns", in.GetNamespace(), "name", in.GetName())
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			oldIn, ok := oldObj.(client.Object)
			if !ok {
				return
			}
			newIn, ok := newObj.(client.Object)
			if !ok {
				return
			}

			oldIn.SetManagedFields(nil)
			newIn.SetManagedFields(nil)

			fmt.Println("upd", "diff", cmp.Diff(oldIn, newIn))
		},
		DeleteFunc: func(obj interface{}) {
			if in, ok := obj.(client.Object); ok {
				fmt.Println("del", "gvk", in.GetObjectKind().GroupVersionKind(), "ns", in.GetNamespace(), "name", in.GetName())
			}
		},
	})

	if err != nil {
		return err
	}

	done := cmd.Context().Done()
	informer.Run(done)

	<-done

	return nil
}

func main() {
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	cmd := &cobra.Command{
		Use:          "wtfk8s",
		Short:        "wtfk8s",
		SilenceUsage: true,
		RunE:         run,
	}

	cmd.Flags().StringVarP(&gvr.Group, "group", "g", "", "")
	cmd.Flags().StringVarP(&gvr.Version, "version", "v", "", "")
	cmd.Flags().StringVarP(&gvr.Resource, "resource", "r", "", "")

	cmd.Flags().StringVar(&kubeConfig, "kubeconfig", os.Getenv("KUBECONFIG"), "")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", corev1.NamespaceAll, "")
	cmd.Flags().StringVarP(&labelSelector, "label-selector", "l", "", "Label selector to filter resources")
	cmd.Flags().StringVarP(&fieldSelector, "field-selector", "f", "", "Field selector to filter resources")
	cmd.Flags().BoolVar(&managedFields, "show-managed-fields", true, "Show managed fields")

	cmd.MarkFlagsRequiredTogether("group", "version", "resource")

	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
