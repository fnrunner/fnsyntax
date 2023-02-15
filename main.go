package main

import (
	"os"

	ctrlcfgv1alpha1 "github.com/fnrunner/fnsyntax/apis/controllerconfig/v1alpha1"
	"github.com/fnrunner/fnsyntax/pkg/ccsyntax"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/yaml"
)

// const yamlFile = "./examples/upf.yaml"
const yamlFile = "./examples/topo4.yaml"

func main() {
	ctrl.SetLogger(zap.New())
	l := ctrl.Log.WithName("fnrun sytax")

	fb, err := os.ReadFile(yamlFile)
	if err != nil {
		l.Error(err, "cannot read file")
		os.Exit(1)
	}
	l.Info("read file")

	ctrlcfg := &ctrlcfgv1alpha1.ControllerConfigSpec{}
	if err := yaml.Unmarshal(fb, ctrlcfg); err != nil {
		l.Error(err, "cannot unmarshal")
		os.Exit(1)
	}
	l.Info("unmarshal succeeded")

	p, result := ccsyntax.NewParser("ctrlName", ctrlcfg)
	if len(result) > 0 {
		l.Error(err, "ccsyntax validation failed", "result", result)
		os.Exit(1)
	}
	l.Info("ccsyntax validation succeeded")

	_, result = p.Parse()
	if len(result) != 0 {
		for _, res := range result {
			l.Error(err, "ccsyntax parsing failed", "result", res)
		}
		os.Exit(1)
	}
	l.Info("ccsyntax parsing succeeded")

	gvks, result := p.GetExternalResources()
	if len(result) > 0 {
		l.Error(err, "ccsyntax get external resources failed", "result", result)
		os.Exit(1)
	}

	// validate if we can resolve the gvr to gvk in the system
	for _, gvk := range gvks {
		l.Info("gvk", "value", gvk)
	}

	for _, image := range p.GetImages() {
		l.Info("image", "imageInfo", image)
	}
}
