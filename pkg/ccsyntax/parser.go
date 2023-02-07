/*
Copyright 2023 Nokia.

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

package ccsyntax

import (
	fnrunv1alpha1 "github.com/fnrunner/fnruntime/apis/fnrun/v1alpha1"
	ctrlcfgv1alpha1 "github.com/fnrunner/fnsyntax/apis/controllerconfig/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Parser interface {
	GetExternalResources() ([]*schema.GroupVersionKind, []Result)
	Parse() (ConfigExecutionContext, []Result)
	GetImages() []*fnrunv1alpha1.Image
}

func NewParser(cfg *ctrlcfgv1alpha1.ControllerConfig) (Parser, []Result) {
	p := &parser{
		cCfg: cfg,
		//d:       dag.NewDAG(),
		//output: map[string]string{},
		l: ctrl.Log.WithName("parser"),
	}
	// add the callback function to record validation results results
	result := p.ValidateSyntax()
	p.rootVertexName = cfg.GetRootVertexName()

	return p, result
}

type parser struct {
	cCfg           *ctrlcfgv1alpha1.ControllerConfig
	rootVertexName string
	l              logr.Logger
}

func (r *parser) Parse() (ConfigExecutionContext, []Result) {
	// initialize the config execution context
	// for each for and watch a new dag is created
	ceCtx, gvar, result := r.init()
	if len(result) != 0 {
		return nil, result
	}
	// resolves the dependencies in the dag
	// step1. check if all dependencies resolve
	// step2. add the dependencies in the dag
	result = r.populate(ceCtx, gvar)
	if len(result) != 0 {
		r.l.Info("populate failed")
		return nil, result
	}
	//fmt.Println("propulate succeded")
	result = r.resolve(ceCtx, gvar)
	if len(result) != 0 {
		r.l.Info("resolve failed")
		return nil, result
	}
	//fmt.Println("resolve succeded")
	result = r.connect(ceCtx, gvar)
	if len(result) != 0 {
		r.l.Info("connect failed")
		return nil, result
	}
	// optimizes the dependncy graph based on transit reduction
	// techniques
	r.transitivereduction(ceCtx)

	ceCtx.Print()
	return ceCtx, nil
}

func (r *parser) transitivereduction(ceCtx ConfigExecutionContext) {
	// transitive reduction for For dag
	for _, od := range ceCtx.GetFOW(FOWFor) {
		for _, dctx := range od {
			dctx.DAG.TransitiveReduction()
			for _, d := range dctx.BlockDAGs {
				d.TransitiveReduction()
			}
		}

	}
	// transitive reduction for Watch dags
	for _, od := range ceCtx.GetFOW(FOWWatch) {
		for _, dctx := range od {
			dctx.DAG.TransitiveReduction()
			for _, d := range dctx.BlockDAGs {
				d.TransitiveReduction()
			}
		}
	}
}
