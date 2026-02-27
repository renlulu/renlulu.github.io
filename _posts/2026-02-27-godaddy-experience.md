---
title: "GoDaddy 踩坑实录：一个域名配置搞了两小时"
date: 2026-02-27 13:00:00 +0800
categories: [随笔]
tags: [godaddy, dns, github-pages, 域名]
---

> 买了个域名，想指向 GitHub Pages。DNS 配置 5 分钟搞定，但 GoDaddy 的各种"贴心"功能让我折腾了两小时。记录一下这次体验，顺便扒一扒 GoDaddy 最近的种种问题。

## 起因

我在 GoDaddy 买了 `renlulu.com`，想指向我的 GitHub Pages 博客。按理说很简单：

1. 加 4 条 A 记录指向 GitHub 的 IP
2. 加 1 条 CNAME 让 `www` 指向 `renlulu.github.io`
3. 在 repo 里放一个 CNAME 文件
4. 完事

实际上，这三步确实在几分钟内就做完了。DNS 也生效了：

```bash
$ dig renlulu.com +short
185.199.108.153
185.199.110.153
185.199.109.153
185.199.111.153
```

四个 GitHub IP，完美。然而打开浏览器一看——

## "Launching Soon"

映入眼帘的不是我的博客，而是一个星空背景的页面，正中间写着大大的 **Launching Soon**，顶部还有一条 GoDaddy 的广告横幅："Buy a domain, then create a website, logo and social posts with GoDaddy's AI tools."

DNS 明明已经指向 GitHub 了，为什么还是 GoDaddy 的页面？

用 `curl -v` 一查，真相了：

```
* Connected to renlulu.com (15.197.225.128) port 443
* Server certificate:
*   issuer: GoDaddy.com, Inc.
```

连接到的 IP 是 `15.197.225.128`（AWS Global Accelerator），证书签发者是 GoDaddy——我的请求根本没到 GitHub，被 GoDaddy 的代理层拦截了。

原来，GoDaddy 在我买域名的时候，**自动**帮我创建了一个免费的 Website Builder 网站，还**自动**把域名绑定上去了。这个 Website 产品在 DNS 之上做了一层代理，所以不管你 A 记录怎么改，流量都会先经过 GoDaddy 的服务器。

## 删不掉的 Website

好，那我把这个 Website 删掉就行了。

进入 GoDaddy 后台，找到 Domain → Products → Website，看到了这个自动创建的 "Coming Soon" 页面。但是——**找不到删除按钮**。

左侧菜单：Dashboard、Domain、Website、Email、Store、Appointments、Marketing、Conversations、Customers、Deals……唯独没有 Settings。

点 Website 展开，只有 Overview 和 Users。没有 Disconnect Domain，没有 Delete Site，没有 Unpublish。

搜了一圈 GoDaddy 的帮助文档，说是在 Website Builder 编辑器里的 Settings 可以操作。点 Edit Website 进去，翻遍了也没找到删除入口。

最后在某个角落终于找到了方法（具体路径我已经记不清了，因为 GoDaddy 的后台 UI 嵌套了至少三层不同风格的界面），成功删掉了那个 Website。

## 删完又出新问题

Website 删了，刷新页面：

```
ERR_TOO_MANY_REDIRECTS
```

重定向循环。再用 `curl` 查一下 DNS——

```bash
$ dig renlulu.com +short
（空）
```

A 记录没了。GoDaddy 在删除 Website 产品的时候，**把我手动添加的 DNS 记录也一起删了**。

于是重新加了一遍 4 条 A 记录，等 DNS 生效，最终确认：

```bash
$ curl -vsk --resolve renlulu.com:443:185.199.108.153 https://renlulu.com
* Server certificate:
*   subject: CN=renlulu.com
*   issuer: Let's Encrypt; CN=R12
```

Let's Encrypt 证书，GitHub.com 服务器。终于好了。

一个本该 5 分钟完成的事情，花了将近两个小时。

## GoDaddy 的问题远不止这些

踩完这个坑，我去搜了一下 GoDaddy 最近的口碑，发现它的问题比我想象的严重得多。

### FTC 安全处罚

2025 年 1 月，美国联邦贸易委员会（FTC）正式[起诉 GoDaddy](https://www.ftc.gov/news-events/news/press-releases/2025/01/ftc-takes-action-against-godaddy-alleged-lax-data-security-its-website-hosting-services)，指控其自 2018 年以来就没有实施基本的安全措施——没有多因素认证、不监控安全威胁、数据传输没有加密。这直接导致了 2019 至 2022 年间的多次重大数据泄露。

讽刺的是，GoDaddy 在这段时间一直对外宣称自己拥有"获奖安全"。

2025 年 5 月 FTC [最终裁定](https://www.ftc.gov/news-events/news/press-releases/2025/05/ftc-finalizes-order-godaddy-over-data-security-failures)：要求 GoDaddy 建立全面信息安全计划，聘请独立第三方评估。**但没有罚款。**

### 续费价格暴涨

这是用户吐槽最多的一条。GoDaddy 用极低的首年价格吸引注册，续费时大幅提价：

| 项目 | 首年价格 | 续费价格 |
|------|---------|---------|
| .com 域名 | $1.99 | $18.99 |
| SSL 证书 | 试用免费 | $119.99/年 |

同样的 .com 域名，Namecheap 续费只要 $8.98/年，Cloudflare 甚至按成本价 $9.15/年。

### DNS 不靠谱

2025 年 9 月 GoDaddy 出过一次大规模 DNS 故障。在 [Cloudflare 社区](https://community.cloudflare.com/t/godaddy-dns-issues/846229)有人反映，把 NS 改到 Cloudflare 之后，GoDaddy 的 parked 页面仍然显示了超过 48 小时。还有人在 Let's Encrypt 论坛报告，买了域名后 DNS 坏了 30 天都没修好。

我今天的经历也印证了这一点——DNS 记录是对的，但流量被 GoDaddy 的代理层劫持。

### 域名停放与回购

有独立游戏开发者[爆料](https://x.com/kchironis/status/1607496457942335489)：域名过期后被 GoDaddy 自己买走，然后假装是第三方持有，让你通过 "GoDaddy Broker" 花 $2,000 - $4,000 买回自己的域名。

2025 年 12 月，Krebs on Security [报道](https://krebsonsecurity.com/2025/12/most-parked-domains-now-serving-malicious-content/)大量停放域名（包括 GoDaddy 托管的）正在投放恶意内容。

### 客服评分

| 平台 | 评分 | 评论数 |
|------|------|--------|
| PissedConsumer | 1.5 / 5 | 531 条 |
| Trustpilot | 大量 1 星 | 数千条 |
| Sitejabber | 低分 | 517 条 |

常见吐槽：取消订单要电话等 45 分钟，承诺回电从不兑现，12 年老客户数据在升级时被永久删除且不赔偿。

### 股价暴跌

GoDaddy 的股价从 2025 年初的高点[暴跌了 65%](https://www.fool.com/investing/2026/02/25/heres-why-godaddy-stock-is-getting-pummeled-today/)，2026 年 2 月创下两年新低。公司还裁掉了 530 名员工（占总人数 8%）。

市场用脚投票了。

## 建议

如果你已经在 GoDaddy 注册了域名，不用急着转走（域名转移有 60 天锁定期，而且 GoDaddy 的域名注册本身没什么问题）。但强烈建议：

1. **把 Nameserver 改到 Cloudflare**（免费）。这样 DNS 解析完全由 Cloudflare 管理，绕过 GoDaddy 的各种代理和停放服务
2. **不要用 GoDaddy 的任何增值服务**——Website Builder、SSL、邮箱、SEO、Marketing，统统不要碰
3. **关闭自动续费**，到期前手动评估是否要续费，或者提前转到 Cloudflare Registrar（按成本价续费）

如果还没注册域名，直接去 [Cloudflare Registrar](https://www.cloudflare.com/products/registrar/) 或 [Namecheap](https://www.namecheap.com/)。省钱省心。
