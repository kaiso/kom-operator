// +build !ignore_autogenerated

// Code generated by operator-sdk. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Http) DeepCopyInto(out *Http) {
	*out = *in
	out.Port = in.Port
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Http.
func (in *Http) DeepCopy() *Http {
	if in == nil {
		return nil
	}
	out := new(Http)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Microservice) DeepCopyInto(out *Microservice) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Microservice.
func (in *Microservice) DeepCopy() *Microservice {
	if in == nil {
		return nil
	}
	out := new(Microservice)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Microservice) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MicroserviceList) DeepCopyInto(out *MicroserviceList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Microservice, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MicroserviceList.
func (in *MicroserviceList) DeepCopy() *MicroserviceList {
	if in == nil {
		return nil
	}
	out := new(MicroserviceList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MicroserviceList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MicroserviceSpec) DeepCopyInto(out *MicroserviceSpec) {
	*out = *in
	if in.Command != nil {
		in, out := &in.Command, &out.Command
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	out.Autoscaling = in.Autoscaling
	in.Routing.DeepCopyInto(&out.Routing)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MicroserviceSpec.
func (in *MicroserviceSpec) DeepCopy() *MicroserviceSpec {
	if in == nil {
		return nil
	}
	out := new(MicroserviceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MicroserviceStatus) DeepCopyInto(out *MicroserviceStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MicroserviceStatus.
func (in *MicroserviceStatus) DeepCopy() *MicroserviceStatus {
	if in == nil {
		return nil
	}
	out := new(MicroserviceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MicroservicesAutoscaling) DeepCopyInto(out *MicroservicesAutoscaling) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MicroservicesAutoscaling.
func (in *MicroservicesAutoscaling) DeepCopy() *MicroservicesAutoscaling {
	if in == nil {
		return nil
	}
	out := new(MicroservicesAutoscaling)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Routing) DeepCopyInto(out *Routing) {
	*out = *in
	if in.Http != nil {
		in, out := &in.Http, &out.Http
		*out = make([]Http, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Routing.
func (in *Routing) DeepCopy() *Routing {
	if in == nil {
		return nil
	}
	out := new(Routing)
	in.DeepCopyInto(out)
	return out
}
