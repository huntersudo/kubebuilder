
### 4.5 用于配置/代码生成的标记
https://cloudnative.to/kubebuilder/reference/markers.html


Kubebuilder 利用一个叫做controller-gen的工具来生成公共的代码和 Kubernetes YAML 文件。 这些代码和配置的生成是由 Go 代码中特殊存在的“标记注释”来控制的。

标记都是以加号开头的单行注释，后面跟着一个标记名称，而跟随的关于标记的特定配置则是可选的。
``` 
// +kubebuilder:validation:Optional
// +kubebuilder:validation:MaxItems=2
// +kubebuilder:printcolumn:JSONPath=".status.replicas",name=Replicas,type=string
```
关于不同类型的代码和 YAML 生成可以查看每一小节来获取详细信息。

####在 KubeBuilder 中生成代码 & 制品
Kubebuilder 项目有两个 make 命令用到了 controller-gen：

- make manifests 用来生成 Kubernetes 对象的 YAML 文件，像CustomResourceDefinitions，WebhookConfigurations 和 RBAC roles。
- make generate 用来生成代码，像runtime.Object/DeepCopy implementations。

查看[生成 CRDs]来获取综合描述。 对应 4-1 部分

#### 标记语法

准确的语法在godocs for controller-tools有描述。
`https://pkg.go.dev/sigs.k8s.io/controller-tools/pkg/markers`

通常，标记可以是：
- Empty (+kubebuilder:validation:Optional)：
  空标记像命令行中的布尔标记位-- 仅仅是指定他们来开启某些行为。
- Anonymous (+kubebuilder:validation:MaxItems=2)：  
  匿名标记使用单个值作为参数。
- Multi-option (+kubebuilder:printcolumn:JSONPath=".status.replicas",name=Replicas,type=string)：
   多选项标记使用一个或多个命名参数。第一个参数与名称之间用冒号隔开，而后面的参数使用逗号隔开。参数的顺序没有关系。有些参数是可选的。

标记的参数可以是字符，整数，布尔，切片，或者 map 类型。 字符，整数，和布尔都应该符合 Go 语法：

````
// +kubebuilder:validation:ExclusiveMaximum=false
// +kubebuilder:validation:Format="date-time"
// +kubebuilder:validation:Maximum=42
````
为了方便，在简单的例子中字符的引号可以被忽略，尽管这种做法在任何时候都是不被鼓励使用的，即便是单个字符：
``` 
// +kubebuilder:validation:Type=string
```
切片可以用大括号和逗号分隔来指定。
``` 
// +kubebuilder:webhooks:Enum={"crackers, Gromit, we forgot the crackers!","not even wensleydale?"}
```
或者，在简单的例子中，用分号来隔开。
``` 
// +kubebuilder:validation:Enum=Wallace;Gromit;Chicken
```
Maps 是用字符类型的键和任意类型的值（有效地map[string]interface{}）来指定的。一个 map 是由大括号（{}）包围起来的，每一个键和每一个值是用冒号（:）隔开的，每一个键值对是由逗号隔开的。
``` 
// +kubebuilder:validation:Default={magic: {numero: 42, stringified: forty-two}}
```


### 4.5.1 CRD 生成
这些标记描述了如何从一系列 Go 类型和包中构建出一个 CRD。而验证标记则描述了实际验证模式的生成。
https://book.kubebuilder.io/reference/markers/crd.html

// +kubebuilder:printcolumn

// +kubebuilder:resource

// +kubebuilder:skipversio

// +kubebuilder:storageversion

// +kubebuilder:subresource:scale

// +kubebuilder:subresource:status

// +kubebuilder:unservedversion

// +groupName

// +kubebuilder:skip

// +versionName


### 4.5.2 CRD Validation
这些标记修改了如何为其修改的类型和字段生成 CRD 验证框架。每个标记大致对应一个 OpenAPI/JSON 模式选项。
https://book.kubebuilder.io/reference/markers/crd-validation.html

// +kubebuilder:default
// +kubebuilder:validation:EmbeddedResource
// +kubebuilder:validation:Enum
// +kubebuilder:validation:ExclusiveMaximum
// +kubebuilder:validation:ExclusiveMinimum
// +kubebuilder:validation:Format
// +kubebuilder:validation:MaxItems
// +kubebuilder:validation:MaxLength
// +kubebuilder:validation:MaxProperties
// +kubebuilder:validation:Maximum
// +kubebuilder:validation:MinItems
// +kubebuilder:validation:MinLength
// +kubebuilder:validation:MinProperties
// +kubebuilder:validation:Minimum
// +kubebuilder:validation:MultipleOf
// +kubebuilder:validation:Optional
// +kubebuilder:validation:Pattern
// +kubebuilder:validation:Required
// +kubebuilder:validation:Schemaless
// +kubebuilder:validation:Type
// +kubebuilder:validation:UniqueItems
// +kubebuilder:validation:XEmbeddedResource
// +nullable
// +optional
// +kubebuilder:validation:Enum
// +kubebuilder:validation:ExclusiveMaximum
// +kubebuilder:validation:ExclusiveMinimum
// +kubebuilder:validation:Format
// +kubebuilder:validation:MaxItems
// +kubebuilder:validation:MaxLength
// +kubebuilder:validation:MaxProperties
// +kubebuilder:validation:Maximum
// +kubebuilder:validation:MinItems
// +kubebuilder:validation:MinLength
// +kubebuilder:validation:MinProperties
// +kubebuilder:validation:Minimum
// +kubebuilder:validation:MultipleOf
// +kubebuilder:validation:Pattern
// +kubebuilder:validation:Type
// +kubebuilder:validation:UniqueItems
// +kubebuilder:validation:XEmbeddedResource
// +kubebuilder:validation:Optional
// +kubebuilder:validation:Required

### 4.5.3  CRD 处理
当你有自定义资源请求时,这些标记有助于 Kubernetes API 服务器控制处理 API。
https://book.kubebuilder.io/reference/markers/crd-processing.html

// +kubebuilder:pruning:PreserveUnknownFields
// +kubebuilder:validation:XPreserveUnknownFields
// +listMapKey
// +listType
// +mapType
// +structType
// +kubebuilder:pruning:PreserveUnknownFields
// +kubebuilder:validation:XPreserveUnknownFields

### 4.5.4 Webhook
这些标记描述了webhook配置如何生成。 使用这些使你的 webhook 描述与实现它们的代码保持一致。
https://book.kubebuilder.io/reference/markers/webhook.html

// +kubebuilder:webhook

### 4.5.5 Object/DeepCopy

这些标记控制何时生成 DeepCopy 和 runtime.Object 实现方法。
https://book.kubebuilder.io/reference/markers/object.html

// +kubebuilder:object:generate:=<bool>  on type

// +kubebuilder:object:root:=<bool>  on type 

// +kubebuilder:object:root:=<bool>  on package 

// +k8s:deepcopy-gen:=<raw>  ========== kubebuilder:object:generate(on package)
// +k8s:deepcopy-gen:=<raw>  ========== kubebuilder:object:generate (on type)

// k8s:deepcopy-gen:interfaces:=<string>  ====  kubebuilder:object:root( on type)


### 4.5.6 RBAC

这些标签会导致生成一个 RBAC 的 ClusterRole。这可以让您描述控制器所需要的权限，以及使用这些权限的代码。
https://cloudnative.to/kubebuilder/reference/markers/rbac.html


// +kubebuilder:rbac on package 













































































































