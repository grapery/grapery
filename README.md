## grapery/执象
#### 期望

    buttom up to build a group to fight back company,org·country.Free man should be always a real exist.
 
---
#### 面向使用场景：（优先级从高到低）
##### 让用户可以长时间的做一件事情

```
算法推荐的坏处就是，每一次用户操作的过程都是不可复现的,用户就只能被动的接受算法的安排；
用户被动的接受信息，很快就对根据算法的推荐失去热情，只有对算法推荐出的东西有少量的惊喜
```

- 每个人都会有各种的表达诉说：
    - 有的是一条接一条的短述，这些短述可以组成一个个的诉说流，可以只自己看，可以分享，也可以给别人订阅
    - 可以每一天拍一些记录的视频，用来真实记录或者让别人了解自己的周遭，可以分享，可以订阅
    - 这些东西可以公开，也可以一个人自己看，或者只分享给一个人看，无论你是暗恋还是莫名的诉说，都可以
    - 日常的阅读分享或者看到好的文章，好的短言分享，比如说长期阅读一部书，分享自己的阅读感受
- 大学许多院系的课程共享和讨论，以及同学之间的大课业协作完成
    - 同方向的同学可以组成一个小组，在小组内建立自己的项目，用来一起协作或者完成课业，课业内可以一起搜集素材，一起互相问答，一起庆祝最后的课业完成
    - 教授或者老师的课程，可以同步到他们自己的课程分类中，这样即使不是本院系的学生，也可以通过他们的课程来学习，虽然互动可能不会很友好，但是至少有课程的结构
    - 大学内的各种学生会，都可以创建自己的项目，这样学生可以参与更多，并且也可以方便新来者发觉新的小组或者学生会
    - 不同院系的同学可以发挥自己的特长，组织在一起，来做一些更新奇的技术，科技，产品，或者展览，艺术，音乐，或者其他
- 大家族内部成员的图片和视频共享，允许家族内部纪念
    - 每年春节的重逢，通过图片或者视频记录，可以在以后回味
    - 家族大事的记录，例如红白喜事或者老人祝寿
    - 夫妻家庭内部的一些日常，例如todo,例如日常送礼，例如家庭长期计划，例如孩子的日常成长，例如孩子的玩闹嬉戏的瞬间


#### 新增功能：



  

#### 各个实体之间的关系

- 用户
    - 用户就是处理器，或者说类似于go语言中的M
    - 用户不可以直接加好友
    - 用户之间没有follow关系，只有用户对组织的follow
    - 用户之间加好友需要有至少(1，暂定)个共同参与的项目或者组织
    - 用户之间好友关系可以解除，可以重复二次提请
    - 用户也有自己的黑名单，不过黑名单仅限于过滤用户自己所拥有的资源，不影响组织和项目内的东西
    - 用户可以点赞项目、关注项目，但不可以follow项目
    - 用户可以加入/退出组织
    
- 组织
    - 组织就是逻辑处理器，类似于go语言中的P,用户需要一个切实的P才可以执行操作
    - 任何一个用户都会有一个默认组织，用来放用户自己的项目，不过是无感知的
    - 用户在默认组织空间中，创建的所有的东西默认都是开放的，想象一下一个人开始从一块空地盖房子，初始大家都知道你在做什么，后面有了房屋之后，就只能看见小院子了
    - 用户可以follow一个组织，但是用户默认的组织是不能被follow的
    - 用户可以加入一个组织，但是用户默认组织是不允许加入的
    - 组织可以设置加入权限，或者完全开放权限，或者隐私设置
    - 组织的总项目数量没有限制，但是一天只允许创建10-50个
    - 组织内可以暂时封禁一个用户，封禁可以组织管理员来做或者驱逐用户可以管理员，也可以投票
    - 组织内有自己的topic话题
    - 组织的话题topic可以继承自外部全局，也可以组织内部新创建（实际由项目来创建）
- 项目
    - 项目就是指令代码，或者说类似于go语言中的G，实际的执行结构体
    - 项目是实际的事件，或者图片，或者短说说，或者音频，或者短视频，或者以上混合
    - 项目必须放在某一个组织下
    - 用户加入一个组织之后，可以参与组织中的任何一个项目
    - 项目有组织创建者和持有者之分
    - 项目可以：正常，已经打包，已经关闭，寻求第三者维护
    - 项目的中的图片，短说说，音频，短视频，都是一个item,每一个item可以是组合的
    - 每一个item可以被单独点赞或者点左中右
    - 项目的评论附属于每一个item
    - 项目可以继承自组织的topic，也可以创建自己的话题topic
    - 每个项目有自己的问题栏,可以搜索问题
    - 每个项目有自己的wiki栏,可以搜索内容，文字的裸搜索，语音和短视频搜索话题topic
    - 搜索是开放式的项目才会被搜索到，但是统计trending会把私有的统计数目
    - 每个项目有自己的TODO栏
---

#### 一些开发规范
```shell
视图层返回全部使用proto,以方便后期使用grpc-gateway
请求数据全部使用post或者put方法，将数据放在body json中，get和delete也可以使用，只限于数据的获取或者删除
```

#### 常用命令
sudo docker run -i -t -e MYSQL_PASSWORD=123456789 -uroot -e MYSQL_ROOT_PASSWORD=123456789  -p 3306:3306  mysql:latest