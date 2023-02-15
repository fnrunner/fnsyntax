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
	"fmt"
	"sync"

	ctrlcfgv1alpha1 "github.com/fnrunner/fnsyntax/apis/controllerconfig/v1alpha1"
	"github.com/fnrunner/fnutils/pkg/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (r *parser) init() (ConfigExecutionContext, GlobalVariable, []Result) {
	i := initializer{
		cec:  NewConfigExecutionContext(r.controllerName),
		gvar: NewGlobalVariable(r.controllerName),
	}

	fnc := &WalkConfig{
		gvkObjectFn:     i.initGvk,
		functionBlockFn: i.initFunctionBlock,
	}
	// walk the config initialaizes the config execution context
	r.walkControllerConfig(fnc)

	return i.cec, i.gvar, i.result

}

type initializer struct {
	cec    ConfigExecutionContext
	gvar   GlobalVariable
	mr     sync.RWMutex
	result []Result
}

func (r *initializer) recordResult(result Result) {
	r.mr.Lock()
	defer r.mr.Unlock()
	r.result = append(r.result, result)
}

func (r *initializer) initGvk(oc *OriginContext, v *ctrlcfgv1alpha1.GvkObject) *schema.GroupVersionKind {
	gvk, err := meta.GetGVKFromRuntimeRawExtension(v.Resource)
	if err != nil {
		r.recordResult(Result{
			OriginContext: oc,
			Error:         err.Error(),
		})
	}
	oc.GVK = gvk
	// initialize execution context for thr for and watch
	if oc.FOWS == FOWFor || oc.FOWS == FOWWatch {
		// initialize the gvk and rootVertex in the execution context
		if err := r.cec.Add(oc); err != nil {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         err.Error(),
			})
		}
	}
	// initialize the output context
	r.gvar.Add(FOWEntry{FOW: oc.FOWS, RootVertexName: oc.VertexName})
	return gvk
}

func (r *initializer) initFunctionBlock(oc *OriginContext, v *ctrlcfgv1alpha1.FunctionElement) {
	if oc.BlockIndex >= 1 {
		// we can only have 1 block index -> only 1 recursion allowed
		r.recordResult(Result{
			OriginContext: oc,
			Error:         fmt.Errorf("a pipeline van only have 1function block %v", *oc).Error(),
		})
	}
	if !v.Function.HasBlock() {
		r.recordResult(Result{
			OriginContext: oc,
			Error:         fmt.Errorf("a function block must have a block %v", *oc).Error(),
		})
	}
	if v.HasBlock() {
		if err := r.cec.AddBlock(oc); err != nil {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         err.Error(),
			})
		}
	}
}
