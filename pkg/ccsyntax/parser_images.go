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

	fnrunv1alpha1 "github.com/fnrunner/fnruntime/apis/fnrun/v1alpha1"
	ctrlcfgv1alpha1 "github.com/fnrunner/fnsyntax/apis/controllerconfig/v1alpha1"
	"github.com/fnrunner/fnutils/pkg/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (r *parser) GetImages() []*fnrunv1alpha1.Image {
	img := &img{
		images: []*fnrunv1alpha1.Image{},
	}
	img.addImageFn = img.addImage

	fnc := &WalkConfig{
		gvkObjectFn: img.getGvk,
		functionFn:  img.getFunctionGvk,
		serviceFn:   img.getFunctionGvk,
	}

	// validate the external resources
	r.walkControllerConfig(fnc)
	return img.images
}

type img struct {
	mrs        sync.RWMutex
	images     []*fnrunv1alpha1.Image
	addImageFn imgAddImageFn
}

type imgAddImageFn func(*fnrunv1alpha1.Image)

func (r *img) addImage(img *fnrunv1alpha1.Image) {
	//fmt.Printf("add gvk: %v \n", gvk)
	r.mrs.Lock()
	defer r.mrs.Unlock()
	found := false
	for _, image := range r.images {
		if image.Name == img.Name {
			return
		}
	}
	if !found {
		r.images = append(r.images, img)
	}
}

func (r *img) getGvk(oc *OriginContext, v *ctrlcfgv1alpha1.GvkObject) *schema.GroupVersionKind {
	gvk := r.getgvk(oc, v.Resource)
	return gvk
}

func (r *img) getgvk(oc *OriginContext, v runtime.RawExtension) *schema.GroupVersionKind {
	gvk, _ := meta.GetGVKFromRuntimeRawExtension(v)
	return gvk
}

func (r *img) getFunctionGvk(oc *OriginContext, v *ctrlcfgv1alpha1.Function) {
	if v.Type == ctrlcfgv1alpha1.ContainerType {
		switch oc.FOWS {
		case FOWService:
			r.addImageFn(&fnrunv1alpha1.Image{
				Name: v.Image,
				Kind: fnrunv1alpha1.ImageKindService,
			})
		default:
			r.addImageFn(&fnrunv1alpha1.Image{
				Name: v.Image,
				Kind: fnrunv1alpha1.ImageKindFunction,
			})
		}
	}
}
