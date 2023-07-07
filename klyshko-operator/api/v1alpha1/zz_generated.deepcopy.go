// +build !ignore_autogenerated

/*
Copyright 2022.

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GeneratorSpec) DeepCopyInto(out *GeneratorSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GeneratorSpec.
func (in *GeneratorSpec) DeepCopy() *GeneratorSpec {
	if in == nil {
		return nil
	}
	out := new(GeneratorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TupleGenerationJob) DeepCopyInto(out *TupleGenerationJob) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TupleGenerationJob.
func (in *TupleGenerationJob) DeepCopy() *TupleGenerationJob {
	if in == nil {
		return nil
	}
	out := new(TupleGenerationJob)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TupleGenerationJob) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TupleGenerationJobList) DeepCopyInto(out *TupleGenerationJobList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TupleGenerationJob, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TupleGenerationJobList.
func (in *TupleGenerationJobList) DeepCopy() *TupleGenerationJobList {
	if in == nil {
		return nil
	}
	out := new(TupleGenerationJobList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TupleGenerationJobList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TupleGenerationJobSpec) DeepCopyInto(out *TupleGenerationJobSpec) {
	*out = *in
	out.Generator = in.Generator
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TupleGenerationJobSpec.
func (in *TupleGenerationJobSpec) DeepCopy() *TupleGenerationJobSpec {
	if in == nil {
		return nil
	}
	out := new(TupleGenerationJobSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TupleGenerationJobStatus) DeepCopyInto(out *TupleGenerationJobStatus) {
	*out = *in
	in.LastStateTransitionTime.DeepCopyInto(&out.LastStateTransitionTime)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TupleGenerationJobStatus.
func (in *TupleGenerationJobStatus) DeepCopy() *TupleGenerationJobStatus {
	if in == nil {
		return nil
	}
	out := new(TupleGenerationJobStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TupleGenerationScheduler) DeepCopyInto(out *TupleGenerationScheduler) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TupleGenerationScheduler.
func (in *TupleGenerationScheduler) DeepCopy() *TupleGenerationScheduler {
	if in == nil {
		return nil
	}
	out := new(TupleGenerationScheduler)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TupleGenerationScheduler) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TupleGenerationSchedulerList) DeepCopyInto(out *TupleGenerationSchedulerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TupleGenerationScheduler, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TupleGenerationSchedulerList.
func (in *TupleGenerationSchedulerList) DeepCopy() *TupleGenerationSchedulerList {
	if in == nil {
		return nil
	}
	out := new(TupleGenerationSchedulerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TupleGenerationSchedulerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TupleGenerationSchedulerSpec) DeepCopyInto(out *TupleGenerationSchedulerSpec) {
	*out = *in
	if in.TupleTypePolicies != nil {
		in, out := &in.TupleTypePolicies, &out.TupleTypePolicies
		*out = make([]TupleTypePolicy, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TupleGenerationSchedulerSpec.
func (in *TupleGenerationSchedulerSpec) DeepCopy() *TupleGenerationSchedulerSpec {
	if in == nil {
		return nil
	}
	out := new(TupleGenerationSchedulerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TupleGenerationSchedulerStatus) DeepCopyInto(out *TupleGenerationSchedulerStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TupleGenerationSchedulerStatus.
func (in *TupleGenerationSchedulerStatus) DeepCopy() *TupleGenerationSchedulerStatus {
	if in == nil {
		return nil
	}
	out := new(TupleGenerationSchedulerStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TupleGenerationTask) DeepCopyInto(out *TupleGenerationTask) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TupleGenerationTask.
func (in *TupleGenerationTask) DeepCopy() *TupleGenerationTask {
	if in == nil {
		return nil
	}
	out := new(TupleGenerationTask)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TupleGenerationTask) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TupleGenerationTaskList) DeepCopyInto(out *TupleGenerationTaskList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TupleGenerationTask, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TupleGenerationTaskList.
func (in *TupleGenerationTaskList) DeepCopy() *TupleGenerationTaskList {
	if in == nil {
		return nil
	}
	out := new(TupleGenerationTaskList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TupleGenerationTaskList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TupleGenerationTaskSpec) DeepCopyInto(out *TupleGenerationTaskSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TupleGenerationTaskSpec.
func (in *TupleGenerationTaskSpec) DeepCopy() *TupleGenerationTaskSpec {
	if in == nil {
		return nil
	}
	out := new(TupleGenerationTaskSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TupleGenerationTaskStatus) DeepCopyInto(out *TupleGenerationTaskStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TupleGenerationTaskStatus.
func (in *TupleGenerationTaskStatus) DeepCopy() *TupleGenerationTaskStatus {
	if in == nil {
		return nil
	}
	out := new(TupleGenerationTaskStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TupleGenerator) DeepCopyInto(out *TupleGenerator) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TupleGenerator.
func (in *TupleGenerator) DeepCopy() *TupleGenerator {
	if in == nil {
		return nil
	}
	out := new(TupleGenerator)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TupleGenerator) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TupleGeneratorList) DeepCopyInto(out *TupleGeneratorList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TupleGenerator, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TupleGeneratorList.
func (in *TupleGeneratorList) DeepCopy() *TupleGeneratorList {
	if in == nil {
		return nil
	}
	out := new(TupleGeneratorList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TupleGeneratorList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TupleGeneratorSpec) DeepCopyInto(out *TupleGeneratorSpec) {
	*out = *in
	out.GeneratorSpec = in.GeneratorSpec
	if in.Supports != nil {
		in, out := &in.Supports, &out.Supports
		*out = make([]TupleTypeSpec, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TupleGeneratorSpec.
func (in *TupleGeneratorSpec) DeepCopy() *TupleGeneratorSpec {
	if in == nil {
		return nil
	}
	out := new(TupleGeneratorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TupleGeneratorStatus) DeepCopyInto(out *TupleGeneratorStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TupleGeneratorStatus.
func (in *TupleGeneratorStatus) DeepCopy() *TupleGeneratorStatus {
	if in == nil {
		return nil
	}
	out := new(TupleGeneratorStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TupleTypePolicy) DeepCopyInto(out *TupleTypePolicy) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TupleTypePolicy.
func (in *TupleTypePolicy) DeepCopy() *TupleTypePolicy {
	if in == nil {
		return nil
	}
	out := new(TupleTypePolicy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TupleTypeSpec) DeepCopyInto(out *TupleTypeSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TupleTypeSpec.
func (in *TupleTypeSpec) DeepCopy() *TupleTypeSpec {
	if in == nil {
		return nil
	}
	out := new(TupleTypeSpec)
	in.DeepCopyInto(out)
	return out
}
