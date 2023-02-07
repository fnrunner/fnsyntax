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
	"sync"

	ctrlcfgv1alpha1 "github.com/fnrunner/fnsyntax/apis/controllerconfig/v1alpha1"
	"github.com/fnrunner/fnutils/pkg/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (r *parser) GetExternalResources() ([]*schema.GroupVersionKind, []Result) {
	er := &er{
		result:    []Result{},
		resources: []*schema.GroupVersionKind{},
	}
	er.resultFn = er.recordResult
	er.addKindFn = er.addGVK

	fnc := &WalkConfig{
		gvkObjectFn: er.getGvk,
		functionFn:  er.getFunctionGvk,
		serviceFn:   er.getFunctionGvk,
	}

	// validate the external resources
	r.walkControllerConfig(fnc)
	return er.resources, er.result
}

type er struct {
	mr        sync.RWMutex
	result    []Result
	resultFn  recordResultFn
	mrs       sync.RWMutex
	resources []*schema.GroupVersionKind
	addKindFn erAddKindFn
}

type erAddKindFn func(*schema.GroupVersionKind)

func (r *er) recordResult(result Result) {
	r.mr.Lock()
	defer r.mr.Unlock()
	r.result = append(r.result, result)
}

func (r *er) addGVK(gvk *schema.GroupVersionKind) {
	//fmt.Printf("add gvk: %v \n", gvk)
	r.mrs.Lock()
	defer r.mrs.Unlock()
	found := false
	for _, resource := range r.resources {
		if resource.Group == gvk.Group &&
			resource.Version == gvk.Version &&
			resource.Kind == gvk.Kind {
			return
		}
	}
	if !found {
		r.resources = append(r.resources, gvk)
	}
}

func (r *er) addGvk(gvk *schema.GroupVersionKind) {
	r.addGVK(gvk)
}

func (r *er) getGvk(oc *OriginContext, v *ctrlcfgv1alpha1.GvkObject) *schema.GroupVersionKind {
	gvk := r.getgvk(oc, v.Resource)
	r.addGvk(gvk)
	return gvk
}

func (r *er) getFunctionGvk(oc *OriginContext, v *ctrlcfgv1alpha1.Function) {
	if v.Input != nil && len(v.Input.Resource.Raw) != 0 {
		gvk := r.getgvk(oc, v.Input.Resource)
		r.addGvk(gvk)
	}
	if v.Type == ctrlcfgv1alpha1.GoTemplateType {
		if len(v.Input.Resource.Raw) != 0 {
			gvk := r.getgvk(oc, v.Input.Resource)
			r.addGvk(gvk)
		}
	}
	for _, v := range v.Output {
		if !v.Internal && len(v.Resource.Raw) != 0 {
			gvk := r.getgvk(oc, v.Resource)
			r.addGvk(gvk)
		}
	}
}

func (r *er) getgvk(oc *OriginContext, v runtime.RawExtension) *schema.GroupVersionKind {
	gvk, err := meta.GetGVKFromRuntimeRawExtension(v)
	if err != nil {
		r.recordResult(Result{
			OriginContext: oc,
			Error:         err.Error(),
		})
	}
	return gvk
}
