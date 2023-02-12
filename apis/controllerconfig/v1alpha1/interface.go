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

package v1alpha1

import (
	"github.com/fnrunner/fnutils/pkg/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (r *ControllerConfigSpec) GetFors() map[string]*GvkObject {
	return r.For
}

func (r *ControllerConfigSpec) GetOwns() map[string]*GvkObject {
	return r.Own
}

func (r *ControllerConfigSpec) GetWatches() map[string]*GvkObject {
	return r.Watch
}

// this function is assumed to be executed after validation
// validate check if the for is present
func (r *ControllerConfigSpec) GetRootVertexName() string {
	for vertexName := range r.For {
		return vertexName
	}
	return ""
}

func (r *ControllerConfigSpec) GetForGvk() ([]*schema.GroupVersionKind, error) {
	gvks, err := r.getGvkList(r.GetFors())
	if err != nil {
		return nil, err
	}
	// there should only be 1 for gvr
	return gvks, nil
}

func (r *ControllerConfigSpec) GetOwnGvks() ([]*schema.GroupVersionKind, error) {
	gvks, err := r.getGvkList(r.GetOwns())
	if err != nil {
		return nil, err
	}
	return gvks, nil
}

func (r *ControllerConfigSpec) GetWatchGvks() ([]*schema.GroupVersionKind, error) {
	gvks, err := r.getGvkList(r.GetWatches())
	if err != nil {
		return nil, err
	}
	return gvks, nil
}

func (r *ControllerConfigSpec) getGvkList(gvrObjs map[string]*GvkObject) ([]*schema.GroupVersionKind, error) {
	gvks := make([]*schema.GroupVersionKind, 0, len(gvrObjs))
	for _, gvrObj := range gvrObjs {
		gvk, err := meta.GetGVKFromRuntimeRawExtension(gvrObj.Resource)
		if err != nil {
			return nil, err
		}
		gvks = append(gvks, gvk)
	}
	return gvks, nil
}

func (r *ControllerConfigSpec) GetServices() map[string]*Function {
	return r.Services
}

func (r *ControllerConfigSpec) GetPipelines() []*Pipeline {
	return r.Pipelines
}

func (r *ControllerConfigSpec) GetPipeline(s string) *Pipeline {
	for _, pipeline := range r.GetPipelines() {
		if pipeline.Name == s {
			return pipeline
		}
	}
	return nil
}

/*
func GetIdxName(idxName string) (string, int) {
	split := strings.Split(idxName, "/")
	idx, _ := strconv.Atoi(split[1])
	return split[0], idx
}
*/

func (v *Function) HasBlock() bool {
	return v.Block.Range != nil || v.Block.Condition != nil
}

func (v *Block) HasRange() bool {
	if v.Range != nil {
		return true
	}
	if v.Condition == nil {
		return false
	}
	return v.Condition.Block.HasRange()
}
