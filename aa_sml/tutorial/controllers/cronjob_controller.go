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

/**
TODO 控制器的工作是确保对于任何给定的对象，世界的实际状态（包括集群状态，以及潜在的外部状态，如 Kubelet 的运行容器或云提供商的负载均衡器）与对象中的期望状态相匹配。
  每个控制器专注于一个根 Kind，但可能会与其他 Kind 交互。
  我们把这个过程称为 reconciling。
  在 controller-runtime 中，为特定种类实现 reconciling 的逻辑被称为 Reconciler。
  Reconciler 接受一个对象的名称，并返回我们是否需要再次尝试（例如在错误或周期性控制器的情况下，如 HorizontalPodAutoscaler）


*/

package controllers

// 我们需要核心 controller-runtime 运行库，以及 client 包和我们的 API 类型包。
import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/go-logr/logr"
	"github.com/robfig/cron"
	kbatch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ref "k8s.io/client-go/tools/reference"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	//"sigs.k8s.io/controller-runtime/pkg/reconcile"

	batchv1 "tutorial/api/v1"
)

// 接下来，kubebuilder 为我们搭建了一个基本的 reconciler 结构。几乎每一个调节器都需要记录日志，并且能够获取对象，所以可以直接使用。

// CronJobReconciler reconciles a CronJob object
type CronJobReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Clock
}

/*
We'll mock out the clock to make it easier to jump around in time while testing,
the "real" clock just calls `time.Now`.
*/
// clock knows how to get the current time.
// It can be used to fake out timing for testing.
type Clock interface {
	Now() time.Time
}
type realClock struct{}

func (_ realClock) Now() time.Time { return time.Now() }

//todo 大多数控制器最终都是在集群上运行，需要rbac的权限， 这里使用 controller-tools RBAC markers，会自动生成yaml文件

//+kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=cronjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=cronjobs/finalizers,verbs=update

// todo Reconcile 实际上是对单个对象进行调谐。我们的 Request 只是有一个名字，但我们可以使用 client 从缓存中获取这个对象。
// 我们返回一个空的结果，没有错误，这就向 controller-runtime 表明我们已经成功地对这个对象进行了调谐，在有一些变化之前不需要再尝试调谐。
// 大多数控制器需要一个日志句柄和一个上下文，所以我们在 Reconcile 中将他们初始化
// 上下文是用来允许取消请求的，也或者是实现 tracing 等功能。它是所有 client 方法的第一个参数。Background 上下文只是一个基本的上下文，没有任何额外的数据或超时时间限制。
// 控制器-runtime通过一个名为logr的库使用结构化的日志记录。正如我们稍后将看到的，日志记录的工作原理是将 键值对 附加到静态消息中。我们可以在我们的调和方法的顶部预先分配一些对，让这些对附加到这个调和器的所有日志行。

var (
	scheduledTimeAnnotation = "batch.tutorial.kubebuilder.io/scheduled-at"
)

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CronJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *CronJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// todo CronJob 控制器的基本逻辑如下:
	//根据名称加载定时任务
	//列出所有有效的 job，更新其状态
	//根据保留的历史版本数清理版本过旧的 job
	//检查当前 CronJob 是否被挂起(如果被挂起，则不执行任何操作)
	//计算 job 下一个定时执行时间
	//如果 job 符合执行时机，没有超出截止时间，且不被并发策略阻塞，执行该 job
	//当任务进入运行状态或到了下一次执行时间， job 重新排队

	_ = context.Background()
	// todo 这是唯一的参数 namespace/name
	log := r.Log.WithValues("cronjob", req.NamespacedName)

	//1: 根据名称加载定时任务
	// 通过 client 获取定时任务。所有 client 方法第一个参数都是 context（用于取消定时任务）作为 第一个参数，把请求对象信息作为最后一个参数。
	// Get 方法例外，它把 NamespacedName 作为中间的第二个参数（大多数方法都没有中间的NamespacedName参数，下文会提到）
	var cronJob batchv1.CronJob
	if err := r.Get(ctx, req.NamespacedName, &cronJob); err != nil {
		log.Error(err, "unable to fetch CronJob")
		//忽略掉 not-found 错误，它们不能通过重新排队修复（要等待新的通知）
		//在删除一个不存在的对象时，可能会报这个错误。
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2: 列出所有有效 job，更新它们的状态
	// 为确保每个 job 的状态都会被更新到，我们需要列出某个 CronJob 在当前命名空间下的所有 job。 和 Get 方法类似，我们可以使用 List 方法来列出 CronJob 下所有的 job。
	// 注意，我们使用变长参数 来映射命名空间和任意多个匹配变量（todo 实际上相当于是建立了一个索引）。
	var childJobs kbatch.JobList
	if err := r.List(ctx, &childJobs, client.InNamespace(req.Namespace), client.MatchingFields{jobOwnerKey: req.Name}); err != nil {
		log.Error(err, "unable to list child Jobs")
		return ctrl.Result{}, err
	}
	// todo 索引的作用
	// 调谐器会获取 cronjob 下的所有 job 以更新它们状态。随着 cronjob 数量的增加，遍历全部 conjob 查找会变的相当低效。为了提高查询效率，这些任务会根据控制器名称建立索引。
	// 缓存后的 job 对象会 被添加上一个 jobOwnerKey 字段。这个字段引用其归属控制器和函数作为索引。在下文中，我们将配置 manager 作为这个字段的索引

	// 查找到所有的 job 后，将其归类为 active，successful，failed 三种类型，同时持续跟踪其 最新的执行情况以更新其状态。牢记，status 值应该是从实际的运行状态中实时获取。
	// 从 cronjob 中读取 job 的状态通常不是一个好做法。应该从每次执行状态中获取。我们后续也采用这种方法。
	// 我们可以检查一个 job 是否已处于 “finished” 状态，使用 status 条件还可以知道它是 succeeded 或 failed 状态。
	// 找出所有有效的 job
	var activeJobs []*kbatch.Job
	var successfulJobs []*kbatch.Job
	var failedJobs []*kbatch.Job
	var mostRecentTime *time.Time // 记录其最近一次运行时间以便更新状态
	/*
		We consider a job "finished" if it has a "Complete" or "Failed" condition marked as true.
		Status conditions allow us to add extensible status information to our objects that other
		humans and controllers can examine to check things like completion and health.
	*/
	isJobFinished := func(job *kbatch.Job) (bool, kbatch.JobConditionType) {
		for _, c := range job.Status.Conditions {
			if (c.Type == kbatch.JobComplete || c.Type == kbatch.JobFailed) && c.Status == corev1.ConditionTrue {
				return true, c.Type
			}
		}

		return false, ""
	}
	// +kubebuilder:docs-gen:collapse=isJobFinished

	/*
		We'll use a helper to extract the scheduled time from the annotation that
		we added during job creation.
	*/
	getScheduledTimeForJob := func(job *kbatch.Job) (*time.Time, error) {
		timeRaw := job.Annotations[scheduledTimeAnnotation]
		if len(timeRaw) == 0 {
			return nil, nil
		}

		timeParsed, err := time.Parse(time.RFC3339, timeRaw)
		if err != nil {
			return nil, err
		}
		return &timeParsed, nil
	}
	// +kubebuilder:docs-gen:collapse=getScheduledTimeForJob

	for i, job := range childJobs.Items {
		_, finishedType := isJobFinished(&job)
		switch finishedType {
		case "": // ongoing
			activeJobs = append(activeJobs, &childJobs.Items[i])
		case kbatch.JobFailed:
			failedJobs = append(failedJobs, &childJobs.Items[i])
		case kbatch.JobComplete:
			successfulJobs = append(successfulJobs, &childJobs.Items[i])
		}

		//将启动时间存放在注释中，当job生效时可以从中读取
		scheduledTimeForJob, err := getScheduledTimeForJob(&job)
		if err != nil {
			log.Error(err, "unable to parse schedule time for child job", "job", &job)
			continue
		}
		if scheduledTimeForJob != nil {
			if mostRecentTime == nil {
				mostRecentTime = scheduledTimeForJob
			} else if mostRecentTime.Before(*scheduledTimeForJob) {
				mostRecentTime = scheduledTimeForJob
			}
		}
	}

	if mostRecentTime != nil {
		cronJob.Status.LastScheduleTime = &metav1.Time{Time: *mostRecentTime}
	} else {
		cronJob.Status.LastScheduleTime = nil
	}
	cronJob.Status.Active = nil
	for _, activeJob := range activeJobs {
		jobRef, err := ref.GetReference(r.Scheme, activeJob)
		if err != nil {
			log.Error(err, "unable to make reference to active job", "job", activeJob)
			continue
		}
		cronJob.Status.Active = append(cronJob.Status.Active, *jobRef)
	}

	// 此处会记录我们观察到的 job 数量。为便于调试，略微提高日志级别。注意，这里没有使用 格式化字符串，使用由键值对构成的固定格式信息来输出日志。这样更易于过滤和查询日志
	log.V(1).Info("job count", "active jobs", len(activeJobs), "successful jobs", len(successfulJobs), "failed jobs", len(failedJobs))

	// 使用收集到日期信息来更新 CRD 状态。和之前类似，通过 client 来完成操作。 针对 status 这一子资源，我们可以使用Status部分的Update方法。
	// status 子资源会忽略掉对 spec 的变更。这与其它更新操作的发生冲突的风险更小， 而且实现了权限分离。
	// 更新状态后，后续要确保状态符合我们在 spec 定下的预期。
	if err := r.Status().Update(ctx, &cronJob); err != nil {
		log.Error(err, "unable to update CronJob status")
		return ctrl.Result{}, err
	}

	// 3: 根据保留的历史版本数清理过旧的 job
	//我们先清理掉一些版本太旧的 job，这样可以不用保留太多无用的 job
	// todo 注意: 删除操作采用的“尽力而为”策略
	// 如果个别 job 删除失败了，不会将其重新排队，直接结束删除操作
	if cronJob.Spec.FailedJobsHistoryLimit != nil {
		sort.Slice(failedJobs, func(i, j int) bool {
			if failedJobs[i].Status.StartTime == nil {
				return failedJobs[j].Status.StartTime != nil
			}
			return failedJobs[i].Status.StartTime.Before(failedJobs[j].Status.StartTime)
		})
		for i, job := range failedJobs {
			if int32(i) >= int32(len(failedJobs))-*cronJob.Spec.FailedJobsHistoryLimit {
				break
			}
			if err := r.Delete(ctx, job, client.PropagationPolicy(metav1.DeletePropagationBackground)); client.IgnoreNotFound(err) != nil {
				log.Error(err, "unable to delete old failed job", "job", job)
			} else {
				log.V(0).Info("deleted old failed job", "job", job)
			}
		}
	}

	if cronJob.Spec.SuccessfulJobsHistoryLimit != nil {
		sort.Slice(successfulJobs, func(i, j int) bool {
			if successfulJobs[i].Status.StartTime == nil {
				return successfulJobs[j].Status.StartTime != nil
			}
			return successfulJobs[i].Status.StartTime.Before(successfulJobs[j].Status.StartTime)
		})
		for i, job := range successfulJobs {
			if int32(i) >= int32(len(successfulJobs))-*cronJob.Spec.SuccessfulJobsHistoryLimit {
				break
			}
			if err := r.Delete(ctx, job, client.PropagationPolicy(metav1.DeletePropagationBackground)); (err) != nil {
				log.Error(err, "unable to delete old successful job", "job", job)
			} else {
				log.V(0).Info("deleted old successful job", "job", job)
			}
		}
	}
	// 4: 检查是否被挂起
	// 如果当前 cronjob 被挂起，不会再运行其下的任何 job，我们将其停止。这对于某些 job 出现异常 的排查非常有用。我们无需删除 cronjob 来中止其后续其他 job 的运行。
	if cronJob.Spec.Suspend != nil && *cronJob.Spec.Suspend {
		log.V(1).Info("cronjob suspended, skipping")
		return ctrl.Result{}, nil
	}

	// 5: 计算 job 下一次执行时间
	//如果 cronjob 没被挂起，则我们需要计算它的下一次执行时间， 同时检查是否有遗漏的执行没被处理

	/*
		### 5: Get the next scheduled run

		If we're not paused, we'll need to calculate the next scheduled run, and whether
		or not we've got a run that we haven't processed yet.
	*/

	/*
		We'll calculate the next scheduled time using our helpful cron library.
		We'll start calculating appropriate times from our last run, or the creation
		of the CronJob if we can't find a last run.

		If there are too many missed runs and we don't have any deadlines set, we'll
		bail so that we don't cause issues on controller restarts or wedges.

		Otherwise, we'll just return the missed runs (of which we'll just use the latest),
		and the next run, so that we can know when it's time to reconcile again.
	*/
	getNextSchedule := func(cronJob *batchv1.CronJob, now time.Time) (lastMissed time.Time, next time.Time, err error) {
		sched, err := cron.ParseStandard(cronJob.Spec.Schedule)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("Unparseable schedule %q: %v", cronJob.Spec.Schedule, err)
		}

		// for optimization purposes, cheat a bit and start from our last observed run time
		// we could reconstitute this here, but there's not much point, since we've
		// just updated it.
		var earliestTime time.Time
		if cronJob.Status.LastScheduleTime != nil {
			earliestTime = cronJob.Status.LastScheduleTime.Time
		} else {
			earliestTime = cronJob.ObjectMeta.CreationTimestamp.Time
		}
		if cronJob.Spec.StartingDeadlineSeconds != nil {
			// controller is not going to schedule anything below this point
			schedulingDeadline := now.Add(-time.Second * time.Duration(*cronJob.Spec.StartingDeadlineSeconds))

			if schedulingDeadline.After(earliestTime) {
				earliestTime = schedulingDeadline
			}
		}
		if earliestTime.After(now) {
			return time.Time{}, sched.Next(now), nil
		}

		starts := 0
		for t := sched.Next(earliestTime); !t.After(now); t = sched.Next(t) {
			lastMissed = t
			// An object might miss several starts. For example, if
			// controller gets wedged on Friday at 5:01pm when everyone has
			// gone home, and someone comes in on Tuesday AM and discovers
			// the problem and restarts the controller, then all the hourly
			// jobs, more than 80 of them for one hourly scheduledJob, should
			// all start running with no further intervention (if the scheduledJob
			// allows concurrency and late starts).
			//
			// However, if there is a bug somewhere, or incorrect clock
			// on controller's server or apiservers (for setting creationTimestamp)
			// then there could be so many missed start times (it could be off
			// by decades or more), that it would eat up all the CPU and memory
			// of this controller. In that case, we want to not try to list
			// all the missed start times.
			starts++
			if starts > 100 {
				// We can't get the most recent times so just return an empty slice
				return time.Time{}, time.Time{}, fmt.Errorf("Too many missed start times (> 100). Set or decrease .spec.startingDeadlineSeconds or check clock skew.")
			}
		}
		return lastMissed, sched.Next(now), nil
	}
	// +kubebuilder:docs-gen:collapse=getNextSchedule

	// 计算出定时任务下一次执行时间（或是遗漏的执行时间）
	missedRun, nextRun, err := getNextSchedule(&cronJob, r.Now())
	if err != nil {
		log.Error(err, "unable to figure out CronJob schedule")
		// 重新排队直到有更新修复这次定时任务调度，不必返回错误
		return ctrl.Result{}, nil
	}
	//上述步骤执行完后，将准备好的请求加入队列直到下次执行， 然后确定这些 job 是否要实际执行
	scheduledResult := ctrl.Result{RequeueAfter: nextRun.Sub(r.Now())} // 保存以便别处复用
	log = log.WithValues("now", r.Now(), "next run", nextRun)

	// 6: 如果 job 符合执行时机，并且没有超出截止时间，且不被并发策略阻塞，执行该 job
	// 如果 job 遗漏了一次执行，且还没超出截止时间，把遗漏的这次执行也补上
	if missedRun.IsZero() {
		log.V(1).Info("no upcoming scheduled times, sleeping until next")
		return scheduledResult, nil
	}

	// 确保错过的执行没有超过截止时间
	log = log.WithValues("current run", missedRun)
	tooLate := false
	if cronJob.Spec.StartingDeadlineSeconds != nil {
		tooLate = missedRun.Add(time.Duration(*cronJob.Spec.StartingDeadlineSeconds) * time.Second).Before(r.Now())
	}
	if tooLate {
		log.V(1).Info("missed starting deadline for last run, sleeping till next")
		// TODO(directxman12): events
		return scheduledResult, nil
	}
	//如果确认 job 需要实际执行。我们有三种策略执行该 job。
	//要么先等待现有的 job 执行完后，在启动本次 job； 或是直接覆盖取代现有的 job；或是不考虑现有的 job，直接作为新的 job 执行。
	//因为缓存导致的信息有所延迟， 当更新信息后需要重新排队。

	// 确定要 job 的执行策略 —— 并发策略可能禁止多个job同时运行
	if cronJob.Spec.ConcurrencyPolicy == batchv1.ForbidConcurrent && len(activeJobs) > 0 {
		log.V(1).Info("concurrency policy blocks concurrent runs, skipping", "num active", len(activeJobs))
		return scheduledResult, nil
	}

	// 直接覆盖现有 job
	if cronJob.Spec.ConcurrencyPolicy == batchv1.ReplaceConcurrent {
		for _, activeJob := range activeJobs {
			// we don't care if the job was already deleted
			if err := r.Delete(ctx, activeJob, client.PropagationPolicy(metav1.DeletePropagationBackground)); client.IgnoreNotFound(err) != nil {
				log.Error(err, "unable to delete active job", "job", activeJob)
				return ctrl.Result{}, err
			}
		}
	}

	//确定如何处理现有 job 后，创建符合我们预期的 job
	/*
		We need to construct a job based on our CronJob's template.  We'll copy over the spec
		from the template and copy some basic object meta.

		Then, we'll set the "scheduled time" annotation so that we can reconstitute our
		`LastScheduleTime` field each reconcile.

		Finally, we'll need to set an owner reference.  This allows the Kubernetes garbage collector
		to clean up jobs when we delete the CronJob, and allows controller-runtime to figure out
		which cronjob needs to be reconciled when a given job changes (is added, deleted, completes, etc).
	*/
	constructJobForCronJob := func(cronJob *batchv1.CronJob, scheduledTime time.Time) (*kbatch.Job, error) {
		// We want job names for a given nominal start time to have a deterministic name to avoid the same job being created twice
		name := fmt.Sprintf("%s-%d", cronJob.Name, scheduledTime.Unix())

		job := &kbatch.Job{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				Name:        name,
				Namespace:   cronJob.Namespace,
			},
			Spec: *cronJob.Spec.JobTemplate.Spec.DeepCopy(),
		}
		for k, v := range cronJob.Spec.JobTemplate.Annotations {
			job.Annotations[k] = v
		}
		job.Annotations[scheduledTimeAnnotation] = scheduledTime.Format(time.RFC3339)
		for k, v := range cronJob.Spec.JobTemplate.Labels {
			job.Labels[k] = v
		}
		if err := ctrl.SetControllerReference(cronJob, job, r.Scheme); err != nil {
			return nil, err
		}

		return job, nil
	}
	// +kubebuilder:docs-gen:collapse=constructJobForCronJob

	// 构建 job
	job, err := constructJobForCronJob(&cronJob, missedRun)
	if err != nil {
		log.Error(err, "unable to construct job from template")
		// job 的 spec 没有变更，无需重新排队
		return scheduledResult, nil
	}

	// ...在集群中创建 job
	if err := r.Create(ctx, job); err != nil {
		log.Error(err, "unable to create Job for CronJob", "job", job)
		return ctrl.Result{}, err
	}

	log.V(1).Info("created Job for CronJob run", "job", job)

	//7: 当 job 开始运行或到了 job 下一次的执行时间，重新排队
	// 最终我们返回上述预备的结果。我们还需重新排队当任务还有下一次执行时。 这被视作最长截止时间——如果期间发生了变更，例如 job 被提前启动或是提前 结束，或被修改，我们可能会更早进行调谐。
	// 当有 job 进入运行状态后，重新排队，同时更新状态
	return scheduledResult, nil
	//
	//
	//
	//return ctrl.Result{}, nil

}

// 启动 CronJob 控制器
// 最后，我们还要完善下我们的启动过程。为了让调谐器可以通过 job 的 owner 值快速找到 job。 我们需要一个索引。声明一个索引键，后续我们可以将其用于 client 的虚拟变量名中，从 job 对象中提取索引值。
//  此处的索引会帮我们处理好 namespaces 的映射关系。所以如果 job 有 owner 值，我们快速地获取 owner 值。
//
//另外，我们需要告知 manager，这个控制器拥有哪些 job。当对应的 job 发生变更或被删除时， 自动调用调谐器对 CronJob 进行调谐。
var (
	jobOwnerKey = ".metadata.controller"
	apiGVStr    = batchv1.GroupVersion.String()
)

// todo 最后，我们将 Reconcile 添加到 manager 中，这样当 manager 启动时它就会被启动。
// 现在，我们只是注意到这个 Reconcile 是在 CronJobs 上运行的。以后，我们也会用这个来标记其他的对象

// SetupWithManager sets up the controller with the Manager.
func (r *CronJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// 此处不是测试，我们需要创建一个真实的时钟
	if r.Clock == nil {
		r.Clock = realClock{}
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &kbatch.Job{}, jobOwnerKey, func(rawObj runtime.Object) []string {
		//获取 job 对象，提取 owner...
		job := rawObj.(*kbatch.Job)
		owner := metav1.GetControllerOf(job)
		if owner == nil {
			return nil
		}
		// ...确保 owner 是个 CronJob...
		if owner.APIVersion != apiGVStr || owner.Kind != "CronJob" {
			return nil
		}

		// ...是 CronJob，返回
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.CronJob{}).
		Owns(&kbatch.Job{}).
		Complete(r)

}

//func (r CronJobReconciler) SetupWithManager(mgr manager.Manager) error {
//
//}
