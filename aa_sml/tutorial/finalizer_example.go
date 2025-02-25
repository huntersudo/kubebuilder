/*

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
// +kubebuilder:docs-gen:collapse=Apache License

/*
First, we start out with some standard imports.
As before, we need the core controller-runtime library, as well as
the client package, and the package for our API types.
*/
package controllers

import (
	"context"

	"k8s.io/kubernetes/pkg/apis/batch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batchv1 "tutorial.kubebuilder.io/project/api/v1"
)

// +kubebuilder:docs-gen:collapse=Imports

/*
By default, kubebuilder will include the RBAC rules necessary to update finalizers for CronJobs.
*/

//+kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=cronjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=cronjobs/finalizers,verbs=update

/*
The code snippet below shows skeleton code for implementing a finalizer.
*/
// todo 要注意的关键点是 finalizers 使对象上的“删除”成为设置删除时间戳的“更新”。
//对象上存在删除时间戳记表明该对象正在被删除。否则，在没有 finalizers 的情况下，删除将显示为协调，缓存中缺少该对象。
//- 如果未删除对象并且未注册 finalizers ，则添加 finalizers 并在 Kubernetes 中更新对象。
//- 如果要删除对象，但 finalizers 列表中仍存在 finalizers ，请执行预删除逻辑并移除 finalizers 并更新对象。
//- 确保预删除逻辑是幂等的。

func (r *CronJobReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("cronjob", req.NamespacedName)

	var cronJob *batchv1.CronJob
	if err := r.Get(ctx, req.NamespacedName, cronJob); err != nil {
		log.Error(err, "unable to fetch CronJob")
		// 在我们删除一个不存在的对象的时，我们会遇到not-found errors这样的报错
		// 我们将暂时忽略，因为不能通过重新加入队列的方式来修复这些错误
		//（我们需要等待新的通知），而且我们可以根据删除的请求来获取它们
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 自定义 finalizer 的名字
	myFinalizerName := "storage.finalizers.tutorial.kubebuilder.io"

	//  检查 DeletionTimestamp 以确定对象是否在删除中
	if cronJob.ObjectMeta.DeletionTimestamp.IsZero() {
		// 如果当前对象没有 finalizer， 说明其没有处于正被删除的状态。
		// 接着让我们添加 finalizer 并更新对象，相当于注册我们的 finalizer。
		if !containsString(cronJob.ObjectMeta.Finalizers, myFinalizerName) {
			cronJob.ObjectMeta.Finalizers = append(cronJob.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Update(context.Background(), cronJob); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		//todo  这个对象将要被删除
		if containsString(cronJob.ObjectMeta.Finalizers, myFinalizerName) {
			// 我们的 finalizer 就在这, 接下来就是处理外部依赖
			if err := r.deleteExternalResources(cronJob); err != nil {
				// 如果无法在此处删除外部依赖项，则返回错误
				//todo 以便可以重试
				return ctrl.Result{}, err
			}
			//如果资源成功删除， 从列表中删除我们的 finalizer 并进行更新。
			cronJob.ObjectMeta.Finalizers = removeString(cronJob.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Update(context.Background(), cronJob); err != nil {
				return ctrl.Result{}, err
			}
		}

		// 当它们被删除的时候停止 reconciliation
		return ctrl.Result{}, nil
	}

	// Your reconcile logic

	return ctrl.Result{}, nil
}

func (r *Reconciler) deleteExternalResources(cronJob *batch.CronJob) error {

	// 删除与 cronJob 相关的任何外部资源
	// 确保删除是幂等性操作且可以安全调用同一对象多次。
	return nil
}

// 辅助函数用于检查
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
从字符串切片中删除字符串。
func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
