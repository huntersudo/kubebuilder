/*
Copyright 2021.

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

package v1

import (
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// todo 我们定义了一个自定义类型来保存我们的并发策略。实际上，它的底层类型是 string，但该类型给出了额外的文档，
//  并允许我们在类型上附加验证，而不是在字段上验证，使验证逻辑更容易复用。

// ConcurrencyPolicy describes how the job will be handled.
// Only one of the following concurrent policies may be specified.
// If none of the following policies is specified, the default one
// is AllowConcurrent.
// +kubebuilder:validation:Enum=Allow;Forbid;Replace
type ConcurrencyPolicy string

const (
	// AllowConcurrent allows CronJobs to run concurrently.
	AllowConcurrent ConcurrencyPolicy = "Allow"

	// ForbidConcurrent forbids concurrent runs, skipping next run if previous
	// hasn't finished yet.
	ForbidConcurrent ConcurrencyPolicy = "Forbid"

	// ReplaceConcurrent cancels currently running job and replaces it with a new one.
	ReplaceConcurrent ConcurrencyPolicy = "Replace"
)

// 我们将使用几个标记（// +comment）来指定额外的元数据。
// 在生成 CRD 清单时，controller-tools 将使用这些数据。我们稍后将看到，controller-tools 也将使用 GoDoc 来生成字段的描述。

// CronJobSpec defines the desired state of CronJob
type CronJobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:MinLength=0

	// The schedule in Cron format, see https://en.wikipedia.org/wiki/Cron.
	Schedule string `json:"schedule"`

	// +kubebuilder:validation:Minimum=0

	// Optional deadline in seconds for starting the job if it misses scheduled
	// time for any reason.  Missed jobs executions will be counted as failed ones.
	// +optional
	StartingDeadlineSeconds *int64 `json:"startingDeadlineSeconds,omitempty"`

	// Specifies how to treat concurrent executions of a Job.
	// Valid values are:
	// - "Allow" (default): allows CronJobs to run concurrently;
	// - "Forbid": forbids concurrent runs, skipping next run if previous run hasn't finished yet;
	// - "Replace": cancels currently running job and replaces it with a new one
	// +optional
	ConcurrencyPolicy ConcurrencyPolicy `json:"concurrencyPolicy,omitempty"`

	// This flag tells the controller to suspend subsequent executions, it does
	// not apply to already started executions.  Defaults to false.
	// +optional
	Suspend *bool `json:"suspend,omitempty"`

	// Specifies the job that will be created when executing a CronJob.
	JobTemplate batchv1beta1.JobTemplateSpec `json:"jobTemplate"`

	// +kubebuilder:validation:Minimum=0

	// The number of successful finished jobs to retain.
	// This is a pointer to distinguish between explicit zero and not specified.
	// +optional
	SuccessfulJobsHistoryLimit *int32 `json:"successfulJobsHistoryLimit,omitempty"`

	// +kubebuilder:validation:Minimum=0

	// The number of failed finished jobs to retain.
	// This is a pointer to distinguish between explicit zero and not specified.
	// +optional
	FailedJobsHistoryLimit *int32 `json:"failedJobsHistoryLimit,omitempty"`
}

// CronJobStatus defines the observed state of CronJob
type CronJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// A list of pointers to currently running jobs.
	// +optional
	Active []corev1.ObjectReference `json:"active,omitempty"`

	// Information when was the last time the job was successfully scheduled.
	// 使用 metav1.Time 而不是 time.Time 来保证序列化的兼容性以及稳定性
	// +optional
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`
}

//TODO CronJobList 只是多个 CronJob 的容器。它是批量操作中使用的种类，像 LIST 。
//通常情况下，我们从不修改任何一个 -- 所有修改都要到 Spec 或者 Status 。
//注释被称为标记。我们将会看到更多的它们，但要知道它们充当额外的元数据， 告诉controller-tools(我们的代码和YAML生成器)额外的信息。
//这个特定的标签告诉 object 生成器这个类型表示一个种类。然后，object 生成器为我们生成 这个所有表示种类的类型一定要实现的runtime.Object接口的实现。

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:storageversion
// TODO 新增V2后，选定V1为存储版本,因为我们将有多个版本，我们将需要标记一个存储版本。 这是一个 Kubernetes API 服务端使用存储我们数据的版本。 我们将选择v1版本作为我们项目的版本。
//  注意如果在存储版本改变之前它们已经被写入那么在仓库中可能存在多个版本 -- 改变存储版本仅仅影响 在改变之后对象的创建/更新。

// CronJob is the Schema for the cronjobs API
type CronJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CronJobSpec   `json:"spec,omitempty"`
	Status CronJobStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CronJobList contains a list of CronJob
type CronJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CronJob `json:"items"`
}

// todo 最后，我们将这个 Go 类型添加到 API 组中。这允许我们将这个 API 组中的类型可以添加到任何Scheme。
func init() {
	SchemeBuilder.Register(&CronJob{}, &CronJobList{})
}
