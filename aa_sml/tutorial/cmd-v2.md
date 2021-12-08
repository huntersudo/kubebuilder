
### https://cloudnative.to/kubebuilder/multiversion-tutorial/tutorial.html

### 教程：多版本 API
大多数项目都是从一个 alpha API 开始的，这个 API 会随着发布版本的不同而变化。 然后，最终大多数项目将会转向更稳定的版本。
一旦你的 API 足够的稳定，你就不能够对它做破坏性的修改。 这就是 API 版本发挥作用的地方。
让我们对 CronJob API spec 做一些改变，确保我们的 CronJob 项目支持所有不同的版本。
如果你还没有准备好，请确保你已经阅读过了基础的 CronJob 教程。

请注意本教程的大部分内容是由形成一个可运行的项目的 literate Go 文件生成的，并且放在了本书的下面源目录下 
docs/book/src/multiversion-tutorial/testdata/project。

### 修改

在 Kubernetes 里，所有版本都必须通过彼此进行安全的往返。这意味着如果我们从版本 1 转换到版本 2，然后回退到版本 1，我们一定会失去一些信息。
因此，我们对 API 所做的任何更改都必须与 v1 中所支持的内容兼容还需要确保我们添加到 v2 中的任何内容在 v1 中都得到支持。
某些情况下，这意味着我们需要向 V1 中添加新的字段，
但是在我们这个例子中，我们不会这么做，因为我们没有添加新的功能。

记住这些，让我们将上面的示例转换为稍微更结构化一点：

v1: 
``` 
schedule: "*/1 * * * *"
```
v2:
``` 
schedule:
  minute: */1
```

###### 新增V2 时，只创建Resource，不创建Controller

```  
[root@master tutorial]# kubebuilder create api --group batch --version v2 --kind CronJob
Create Resource [y/n]
y
Create Controller [y/n]
n
Writing kustomize manifests for you to edit...
Writing scaffold for you to edit...
api/v2/cronjob_types.go
Update dependencies:
$ go mod tidy
Running make:
$ make generate
/root/golang/src/tutorial/bin/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."
Next: implement your new API and generate the manifests (e.g. CRDs,CRs) with:
$ make manifests

```

复制V1的 cronjob_types.go 到v2 ，
- 修改其中需要修改的字段： // todo 与v1 相比，除了将 schedule 字段更改为一个新类型外，我们将基本上保持 spec 不变。
- 选定存储版本：
  // +kubebuilder:storageversion
  // TODO 新增V2后，选定V1为存储版本,因为我们将有多个版本，我们将需要标记一个存储版本。 这是一个 Kubernetes API 服务端使用存储我们数据的版本。 我们将选择v1版本作为我们项目的版本。
  //  注意如果在存储版本改变之前它们已经被写入那么在仓库中可能存在多个版本 -- 改变存储版本仅仅影响 在改变之后对象的创建/更新。

#### 现在我们已经准备好了类型，接下来需要设置转换。 Hubs, spokes, 和其他的 wheel metaphors

由于我们现在有两个不同的版本，用户可以请求任意一个版本，我们必须定义一种在版本之间进行转换的方法。
对于 CRD ，这是通过使用 Webhook 完成的，类似我们在基础中定义 webhooks 教程的默认设置和验证一样。
像以前一样，控制器运行时将帮助我们将所有细节都连接在一起，而我们只需实现本身的转换即可。
在执行此操作之前，我们需要了解控制器运行时如何处理版本的。即：

##### 任意两个版本间转换的不足之处

定义转换的一种简单方法可能是定义转换函数如何可以在我们的每个版本之间进行转换。
然后，只要我们需要进行转换的时候，我们只需要查找适当的函数，然后调用它就可以执行转换。当我们只有两个版本时，这可以正常工作，
但是如果我们有4个版本的时候，或者更多的时候该怎么办？那将会有很多转换功能。

相反，控制器运行时会根据 “hub 和 spoke” 模型-我们将一个版本标记为“hub”，而所有其他版本只需定义为与 hub 之间的来源即可：

`https://cloudnative.to/kubebuilder/multiversion-tutorial/conversion-concepts.html`


如果我们必须在两个 non-hub 之间进行转换，则我们首先要进行转换到这个 hub 对应的版本，然后再转换到我们所需的版本：

这样就减少了我们所需定义转换函数的数量，其实就是在模仿 Kubernetes 内部实际的工作方式。

##### 与 Webhooks 有什么关系？

当 API 客户端（例如 kubectl 或你的控制器）请求特定的版本的资源，Kubernetes API 服务器需要返回该版本的结果。
但是，该版本可能不匹配 API 服务器实际存储的版本。

在这种情况下，API 服务器需要知道如何在所需的版本和存储的版本之间进行转换。
由于转换不是 CRD 内置的，于是 Kubernetes API 服务器通过调用 Webhook 来执行转换。
对于 KubeBuilder ，跟我们上面讨论一样，Webhook 通过控制器运行时来执行 hub-and-spoke 的转换。

现在我们有了向下转换的模型，我们就可以实现转换操作了。


#### 实现转换 
https://cloudnative.to/kubebuilder/multiversion-tutorial/conversion.html

采用的转换模型已经就绪，就可以开始实现转换函数了。 
我们将这些函数放置在 cronjob_conversion.go 文件中，cronjob_conversion.go 文件和 cronjob_types.go 文件同目录，以避免我们主要的类型文件和额外的方法产生混乱。

##### Hub...
首先，我们需要实现 hub 接口。我们会选择 v1 版本作为 hub 的一个实现：
实现 hub 方法相当容易 -- 我们只需要添加一个叫做 Hub() 的空方法来作为一个 标记。我们也可以将这行代码放到 cronjob_types.go 文件中。
``` 
vi project/api/v1/cronjob_conversion.go
// Hub 标记这个类型是一个用来转换的 hub。
func (*CronJob) Hub() {}
```
##### 然后 Spokes
然后，我们需要实现我们的 spoke 接口，例如 v2 版本：
我们的 “spoke” 版本需要实现 Convertible 接口。
顾名思义，它需要实现 ConvertTo 从（其它版本）向 hub 版本转换，ConvertFrom 实现从 hub 版本转换到（其他版本）。

``` 
project/api/v2/cronjob_conversion.go

ConvertTo(dst Hub) error
ConvertFrom(src Hub) error
	
```
现在我们的转换方法已经就绪，我们要做的就是启动我们的 main 方法来运行 webhook。

##### 设置 webhook
我们的 conversion 已经就位，所以接下来就是告诉 controller-runtime 关于我们的 conversion。
通常，我们通过运行

之前已经有了，看内容好像没啥区别

```  
kubebuilder create webhook --group batch --version v1 --kind CronJob --conversion
```
根据  docs/book/src/multiversion-tutorial/testdata/project/api/v2/cronjob_webhook.go
应该要生成v2的webhook，直接默认的
``` 
kubebuilder create webhook --group batch --version v2 --kind CronJob --conversion
```

来搭建起 webhook 设置。然而，当我们已经创建好默认和验证过的 webhook 时，我们就已经设置好 webhook。

##### 以及 main.go
所有都已经设置准备好！接下来要做的只有测试我们的 webhook。

##### 部署和测试 

在测试版本转换之前，我们需要在 CRD 中启用转换：
Kubebuilder 在 config 目录下生成禁用 webhook bits 的 Kubernetes 清单。要启用它们，我们需要：
- 在 config/crd/kustomization.yaml 文件启用 patches/webhook_in_<kind>.yaml 和 patches/cainjection_in_<kind>.yaml。
- 在 config/default/kustomization.yaml 文件的 bases 部分下启用 ../certmanager 和 ../webhook 目录。
- 在 config/default/kustomization.yaml 文件的 patches 部分下启用 manager_webhook_patch.yaml。
- 在 config/default/kustomization.yaml 文件的 CERTMANAGER 部分下启用所有变量。

此外，我们需要将 CRD_OPTIONS 变量设置为 "crd"，删除 trivialVersions 选项（这确保我们实际 为每个版本生成验证，而不是告诉 Kubernetes 它们是一样的）：
``` 
CRD_OPTIONS ?= "crd"
```
--这东西没找到  


现在，我们已经完成了所有的代码更改和清单，让我们将其部署到集群并对其进行测试。
你需要安装 cert-manager（0.9.0+ 版本）， 除非你有其他证书管理解决方案。Kubebuilder 团队已在 0.9.0-alpha.0 版本中测试了本教程中的指令。
https://cloudnative.to/kubebuilder/cronjob-tutorial/cert-manager.html

一旦所有的证书准备妥当后, 我们就可以运行 make install deploy（和平常一样）将所有的 bits（CRD, controller-manager deployment）部署到集群上。

##### 测试
一旦启用了转换的所有 bits 都在群集上运行，我们就可以通过请求不同的版本来测试转换。
我们将基于 v1 版本制作 v2 版本（将其放在 config/samples 下）

```` 
kubectl apply -f config/samples/batch_v2_cronjob.yaml

kubectl get cronjobs.v2.batch.tutorial.kubebuilder.io -o yaml
apiVersion: batch.tutorial.kubebuilder.io/v2
kind: CronJob
metadata:
  name: cronjob-sample
spec:
  schedule:
    minute: "*/1"
  startingDeadlineSeconds: 60
  concurrencyPolicy: Allow # explicitly specify, but Allow is also default.
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: hello
            image: busybox
            args:
            - /bin/sh
            - -c
            - date; echo Hello from the Kubernetes cluster
          restartPolicy: OnFailure

kubectl get cronjobs.v1.batch.tutorial.kubebuilder.io -o yaml

apiVersion: batch.tutorial.kubebuilder.io/v1
kind: CronJob
metadata:
  name: cronjob-sample
spec:
  schedule: "*/1 * * * *"
  startingDeadlineSeconds: 60
  concurrencyPolicy: Allow # explicitly specify, but Allow is also default.
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: hello
            image: busybox
            args:
            - /bin/sh
            - -c
            - date; echo Hello from the Kubernetes cluster
          restartPolicy: OnFailure
          
````
两者都应填写，并分别看起来等同于的 v2 和 v1 示例。注意，每个都有不同的 API 版本。

最后，如果稍等片刻，我们应该注意到，`即使我们的控制器是根据 v1 API 版本编写的，我们的 CronJob 仍在继续协调。`

#### kubectl 和首选版本

当我们从 Go 代码访问 API 类型时，我们会通过使用该版本的 Go 类型（例如 batchv2.CronJob）来请求特定版本。

你可能已经注意到，上面对 kubectl 的调用与我们通常所做的看起来有些不同 —— 即，它指定了一个 group-version-resource 而不是一个资源。
kubectl get cronjobs.v1.batch.tutorial.kubebuilder.io -o yaml

当我们运行 kubectl get cronjob 时, kubectl 需要弄清楚映射到哪个 group-version-resource。
为此，它使用 discovery API 来找出 cronjob 资源的首选版本。对于 CRD， 这或多或少是最新的稳定版本（具体细节请参阅 CRD 文档)。

随着我们对 CronJob 的更新, 意味着 kubectl get cronjob 将获取 batch/v2 group-version。
如果我们想指定一个确切的版本，可以像上面一样使用 kubectl get resource.version.group。

你应该始终在脚本中使用完全合格的 group-version-resource 语法。 kubectl get resource 是为人类、有自我意识的机器人和其他能够理解新版本的众生设计的。
kubectl get resource.version.group 用于其他的一切。













