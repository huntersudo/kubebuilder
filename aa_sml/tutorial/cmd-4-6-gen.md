

###4.5  controller-gen CLI
https://book.kubebuilder.io/reference/controller-gen.html
https://cloudnative.to/kubebuilder/reference/controller-gen.html

KubeBuilder 使用了一个称为 controller-gen 用于生成通用代码和 Kubernetes YAML。 代码和配置的生成规则是被 Go 代码中的一些特殊标记注释控制的。
controller-gen 由不同的“generators”(指定生成什么)和“输出规则”(指定如何以及在何处输出结果)。
两者都是通过指定的命令行参数配置的，更详细的说明见 标记格式化。

```` 
controller-gen paths=./... crd:trivialVersions=true rbac:roleName=controller-perms output:crd:artifacts:config=config/crd/bases
````
生成的 CRD 和 RBAC YAML 文件默认存储在config/crd/bases目录。 RBAC 规则默认输出到(config/rbac)。 主要考虑到当前目录结构中的每个包的关系。 (按照 go ... 的通配符规则)。


### 生成器
每个不同的生成器都是通过 CLI 选项配置的。controller-gen 一次运行也可以指定多个生成器。

// +webhook

// +schemapatch

// +rbac

// +object

// +crd

### 输出规则

输出规则配置给定生成器如何输出其结果。 默认是一个全局 fallback 输出规则(指定为 output:<rule>)，
另外还有 per-generator 的规则(指定为output:<generator>:<rule>)，会覆盖掉 fallback 规则。

默认规则

如果没有手动指定 fallback 规则，默认的 per-generator 将被使用，生成的 YAML 将放到 config/<generator>相应目录，代码所在的位置不变。
对于每个生成器来说，默认的规则等价于output:<generator>:artifacts:config=config/<generator>。
指定 fallback 规则后，将使用该规则代替默认规则。
例如，如果你指定crd rbac:roleName=controller-permsoutput:crd:stdout，你将在标准输出中获得 CRD，在config/rbac目录得到 rbac 规则。
如果你要添加全局规则，例如crdrbac:roleName=controller-perms output:crd:stdout output:none，CRD 会被重定向到终端输出，其他被重定向到 /dev/null，因为我们已经明确指定了 fallback 。

为简便起见，每个生成器的输出规则(output:<generator>:<rule>)默认省略。 相当于这里列出的全局备用选项。

// +output:artifacts:code=<string>,config=<string> on package
// +output:dir:=<string> on package 
// +output:none on package
// +output:stdout on package 

### 其他选项

// +paths:=<[]string>   on packge 




### 4.7 开启 shell 自动补全
https://cloudnative.to/kubebuilder/reference/completion.html

### 4.9 在集成测试中使用 envtest
在集成测试中使用 envtest
https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest

controller-runtime 提供 envtest (godoc)，这个包可以帮助你为你在 etcd 和 Kubernetes API server 中设置并启动的 controllers 实例来写集成测试，不需要 kubelet，controller-manager 或者其他组件。

可以根据以下通用流程在集成测试中使用 envtest：
``` 
import sigs.k8s.io/controller-runtime/pkg/envtest

//指定 testEnv 配置
testEnv = &envtest.Environment{
    CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
}

//启动 testEnv
cfg, err = testEnv.Start()

//编写测试逻辑

//停止 testEnv
err = testEnv.Stop()
```

kubebuilder 为你提供了 testEnv 的设置和清除模版，在生成的 /controllers 目录下的 ginkgo 测试套件中。

测试运行中的 Logs 以 test-env 为前缀。

配置你的测试控制面

你可以在你的集成测试中使用环境变量和/或者标记位来指定 api-server 和 etcd 设置。

环境变量
- USE_EXISTING_CLUSTER: bool 可以指向一个已存在 cluster 的控制面，而不用设置一个本地的控制面。
- KUBEBUILDER_ASSETS: 将集成测试指向一个包含所有二进制文件（api-server，etcd 和 kubectl）的目录。
- TEST_ASSET_KUBE_APISERVER, TEST_ASSET_ETCD, TEST_ASSET_KUBECTL: 
和 KUBEBUILDER_ASSETS 相似，但是更细一点。指示集成测试使用非默认的二进制文件。这些环境变量也可以被用来确保特定的测试是在期望版本的二进制文件下运行的。
- KUBEBUILDER_CONTROLPLANE_START_TIMEOUT 和 KUBEBUILDER_CONTROLPLANE_STOP_TIMEOUT: 
time.ParseDuration 支持的持续时间的格式,指定不同于测试控制面（分别）启动和停止的超时时间；任何超出设置的测试都会运行失败。
- KUBEBUILDER_ATTACH_CONTROL_PLANE_OUTPUT: bool ,设置为 true 可以将控制面的标准输出和标准错误贴合到 os.Stdout 和 os.Stderr 上。这种做法在调试测试失败时是非常有用的，因为输出包含控制面的输出。

### 标记位 

下面是一个在你的集成测试中通过修改标记位来启动 API server 的例子，和 envtest.DefaultKubeAPIServerFlags 中的默认值相对比：

``` 
var _ = BeforeSuite(func(done Done) {
    Expect(os.Setenv("TEST_ASSET_KUBE_APISERVER", "../testbin/bin/kube-apiserver")).To(Succeed())
    Expect(os.Setenv("TEST_ASSET_ETCD", "../testbin/bin/etcd")).To(Succeed())
    Expect(os.Setenv("TEST_ASSET_KUBECTL", "../testbin/bin/kubectl")).To(Succeed())

    logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
    testenv = &envtest.Environment{}

    _, err := testenv.Start()
    Expect(err).NotTo(HaveOccurred())

    close(done)
}, 60)

var _ = AfterSuite(func() {
    Expect(testenv.Stop()).To(Succeed())

    Expect(os.Unsetenv("TEST_ASSET_KUBE_APISERVER")).To(Succeed())
    Expect(os.Unsetenv("TEST_ASSET_ETCD")).To(Succeed())
    Expect(os.Unsetenv("TEST_ASSET_KUBECTL")).To(Succeed())

})
```

``` 
customApiServerFlags := []string{
    "--secure-port=6884",
    "--admission-control=MutatingAdmissionWebhook",
}

apiServerFlags := append([]string(nil), envtest.DefaultKubeAPIServerFlags...)
apiServerFlags = append(apiServerFlags, customApiServerFlags...)

testEnv = &envtest.Environment{
    CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
    KubeAPIServerFlags: apiServerFlags,
}
```

#### 测试注意事项

除非你在使用一个已存在的 cluster，否则需要记住在测试内容中没有内置的 controllers 在运行。
在某些方面，测试控制面会表现的和“真实” clusters 有点不一样，这可能会对你如何写测试有些影响。
一个很常见的例子就是垃圾回收；因为没有 controllers 来监控内置的资源，对象是不会被删除的，即使设置了 OwnerReference。

为了测试删除生命周期是否工作正常，要测试所有权而不是仅仅判断是否存在。比如：

``` 
expectedOwnerReference := v1.OwnerReference{
    Kind:       "MyCoolCustomResource",
    APIVersion: "my.api.example.com/v1beta1",
    UID:        "d9607e19-f88f-11e6-a518-42010a800195",
    Name:       "userSpecifiedResourceName",
}
Expect(deployment.ObjectMeta.OwnerReferences).To(ContainElement(expectedOwnerReference))
```

















