---
title: Crypto - Group Dynamics
description: 2021 | ZKHACK | Crypto
---

- [1. Puzzle Description](#1-puzzle-description)
- [2. Preliminery](#2-preliminery)
  - [Chinese Reminder Theorem](#chinese-reminder-theorem)
  - [Groth16 Trusted Setup](#groth16-trusted-setup)
  - [Prime Order Subgroups and Elliptic Curve Cofactors](#prime-order-subgroups-and-elliptic-curve-cofactors)
    - [例子](#例子)
  - [Pohlig-Hellman Algorithm](#pohlig-hellman-algorithm)
    - [Babystep-Giantstep Algorithm](#babystep-giantstep-algorithm)
    - [Pohlig-Hellman Algorithm](#pohlig-hellman-algorithm-1)
    - [优化](#优化)
- [3. Solution](#3-solution)


## 1. Puzzle Description

> If it's small, it should not be here

Alice has computed a trusted setup for a Groth16 proof scheme.
She decided to use a 128-bit long secret, and she swears that she does not know the secret s needed to get this setup.
The trusted setup is constructed as follows using two additional scalars $\alpha$ and $\beta$:
* $[s^i] G_1$  for $0 \le i \le 62$,
* $[\alpha s^i] G_1$  for $0 \le  i \le  31$,
* $[\beta s^i] G_1$  for $0 \le  i \le  31$,
* $[s^i] G_2$  for $0 \le  i \le  31$.

## 2. Preliminery

### Chinese Reminder Theorem

中国剩余定理（CRT）是一个非常经典的求解一元线性同余方程组的算法。

给定一元线性同余方程组（其中模数两两互质）：

$$
\begin{cases}
x\equiv a_1\pmod{n_1}\\
x\equiv a_2\pmod{n_2}\\
\dots\\
x\equiv a_k\pmod{n_k}
\end{cases}
$$

算法流程：

1. 首先计算所有模数乘积 $n=n_1 n_2\dots n_k$
2. 对于第 $i$ 个方程:
   
   a. $m_i=\frac{n}{n_i}$

   b. 计算 $m_i$ 在模 $n_i$ 意义下的逆元 $m_i^{-1}$

3. 方程组的解为： $x=\sum_{i=1}^k{a_im_i m_i^{-1}}\pmod{n}$

这个算法的 intuition 是：让求和的每一项只让其在模 $n_i$ 时取 $a_i$，让每一项在模 $n_j(j\ne i)$ 时等于 $0$。 $m_i m_i^{-1}$ 的意义就是满足了这种 intuition。

时间复杂度主要是 $k$ 次求逆元： $O(k\log n)$

### Groth16 Trusted Setup

Groth16 是一个非常有名的 zkSNARK 协议，以其简短的 proof size 和快速的 verification time （均与电路规模无关）被用于各种小规模电路场景。

为了实现上述优势，Groth16 需要进行一个 Trusted Setup 来生成一些参数供 Prover 和 Verifier 使用。之所以叫 Trusted ，意思是这些参数中存在一个陷门 trapdoor，这个秘密值不能让任何人知道，否则整个协议将没有任何安全性，恶意的证明者可以为虚假的 claim 伪造证明。因此，一个 bad trusted setup 将会导致一个极不安全的协议。

Groth16 Trusted Setup 的安全性基于群上的离散对数问题（Discrete Logarithm Problem, DLP），即给定两个群元素 $(g, g^s)\in \mathbb{G}$，计算出 $s$ 是困难的。本文使用乘法记号 $g^s$ 来表示群的运算，题面使用的是加法记号，二者是等价的。

更具体地，为了支持电路规模为 $n$ 的证明， Groth16 Trusted Setup 需要生成 $(4n-1)$ 个 $\mathbb{G}_1$ 群元素，以及 $n$ 个 $\mathbb{G}_2$ 群元素，具体元素构成如题面所示。其中这两个群分别来自两条曲线的循环子群，满足 Pairing 性质。

但是，DLP 并不是在任何情况下都是难解决的，它在某些情况下是容易解决的，这就是本题考察的点。我第一反应是这篇 [blog](https://kel.bz/post/pohlig/)，介绍了如何求解 smooth integer 阶群的 DLP。

### Prime Order Subgroups and Elliptic Curve Cofactors

To make us on the same page, 首先介绍一下有限域上椭圆曲线各个参数的含义。

有限域上椭圆曲线定义为： $E(\mathbb{F}_p): y^2=x^3+ax+b\pmod p$ ，其中 $p$ 就是有限域的模数，它定义了有限域 $\mathbb{F}_p$ ，而且其他 4 个参数 $x, y, a, b$ 均是有限域中的元素。这个有限域我们称之为域（Base Field）。

把有限域中所有的元素代入 $x$，我们可以得到一个椭圆曲线点坐标集合 $\{(x_0, y_0), (x_1, y_1)...\}$，这个集合构成一个群，群元素满足椭圆曲线上点的加法运算（可以延伸至倍点运算）。由于椭圆曲线上点的加法运算满足交换律，所以这个群是一个**阿贝尔群**（Abelian Group）。我们可以使用 Schoof 算法有效地计算点的个数，也就是这个群的阶。

一般而言，我们需要找到一个有限域上椭圆曲线这个点集所构成的阿贝尔群满足某些好的性质，这样才能满足构建密码学方案的需求。比如这个群有一个大素阶数的循环子群（Cyclic Subgroup），就是一个很好的性质。通常偏好选择阶为大素数的循环子群，因为这样的选择提供了良好的安全性。素数阶的循环子群意味着离散对数问题（DLP）在该群中更难解决。在这个上下文中，我们挑选的循环子群是素数阶子群（Prime Order Subgroup），而非素数的幂阶子群。

假设大素数循环子群的阶为 $r$ ，它也可以形成一个有限域 $\mathbb{F}_r$，我们称之为标量域（Scalar Field）。

> 注意：循环群的阶不一定是素数，理论上还有可能循环子群阶为素数的幂，虽然这依然是循环群，但素数的幂作为群的阶不如素数阶那样普遍，主要是因为素数阶提供了更直接的安全性保证。

我们还需要了解什么是 cofactor？我们已经知道有限域上椭圆曲线的点构成一个阿贝尔群，这个群阶我们可以通过 Schoof 算法有效算出。假设群阶为 $q$，根据算术基本定理（Fundamental Theorem of Arithmetic），一个大于 1 的正整数可以表示成若干个质数乘积的形式。考虑这个群的某个阶为 $r$ 的子群（显然 $r | q$），有 $q = hr$，这个 h 就被称之为子群的 cofactor。


#### 例子

Bitcoin使用的椭圆曲线是 secp256k1，它的曲线参数可以在[这里](https://neuromancer.sk/std/secg/secp256k1)找到。

它的 Base field 为 $\mathbb{F}_p(p=0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffefffffc2f)$，Scalar field 为 $\mathbb{F}_r(r=0xfffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141)$，cofactor 为 0x1。这意味着 Base field 上的椭圆曲线 $y^2\equiv x^3+7\pmod{p}$
 一共有 $r * cofactor$ 个点，并且最大循环子群就是它本身。

 > 有些上下文也会把 base field 记作 $\mathbb{F}_q$，而把 $\mathbb{F}_p$ 统一用作表示素数有限域。比如 Arkwork 下 ark-ff 就

### Pohlig-Hellman Algorithm

有了上述概念基础，我们来介绍如何使用 [Pohlig-Hellman](https://en.wikipedia.org/wiki/Pohlig%E2%80%93Hellman_algorithm) 算法来攻击 cofactor 较小的循环子群，这种攻击也被称之为 `Subgroup Confinement Attacks` ，它利用了群阶是 smooth integer 的事实形成了求解 DLP 的攻击优势。

WLOG，假设生成元为 $g$ 的循环群 $\langle g\rangle$ 阶为 $r=\prod_{i=1}^n p_i^{e_i}$， $p_i$ 是质数. 给定 $g, h = g^x$ （其中 $x\in Z_r$），我们希望计算 $x$ 的值。

#### Babystep-Giantstep Algorithm

大步小步算法是通用的求解离散对数的方法，无前置要求。它基于  `meet-in-the-middle` 思想。

给定 $g^x=h$，选取一个大小接近于 $\sqrt{r}$ 的数 $T$，则 $x$ 可以写成 $x=aT+b$， $a,b$未知 $(a,b \le T)$。 

那么原式可以写作 $g^{aT+b}=h$，即 $g^{aT}=hg^{-b}$。

注意到等式两边分别需要枚举 $a,b$ 次。那么爆破所有可能的 $a,b$，寻找碰撞，进而得到 $aT+b=x$。

这个算法的时间复杂度为 $O(\sqrt{r})$。

#### Pohlig-Hellman Algorithm

有没有更高效的算法呢？有！

我们要求 $x\pmod r$，考虑中国剩余定理 CRT，如果我们知道了 $r$ 的质因数分解 $r=\prod_{i=1}^n p_i^{e_i}$，分别计算出 $x\pmod{p_i^{e_i}}$ ，那么我们就可以快速求出 $x\pmod r$.

于是问题规约成了计算 $a_i=x\pmod{p_i^{e_i}}$，简记 $P_i=p_i^{e_i}$，则有： $x=k_i P_i + a_i$

那么如何计算 $a_i$呢？注意到：

$$
\begin{aligned}
h^{r/P_i}&=g^{(r/P_i)x} \\
&=g^{r(k_i P_i + a_i)/P_i}\\
&=g^{ra_i/P_i}\\
&=(g^{r/P_i})^{a_i}
\end{aligned}
$$

注意到等式左边 $h'=h^{r/P_i}$ 和右边 $g'=g^{r/P_i}$ 都可以直接计算。于是问题转变成了一个新的离散对数问题 $h'=g'^{a_i}$。

于是，我们可以把原先的 $h=g^x$ 问题规约到求解 $n$ 个 $h'=g'^{a_i}$ 问题上。注意新的离散对数问题的 $g'$ 的阶为 $P_i$ ，不再是 $r$。

这个算法时间复杂度为； $O(\sum_{i=1}^n{ \sqrt{p_i^{e_i}}+\log r})$

上面的式子有个严重的问题，如果 $r=p^e$ 那相当于优化都没做，时间复杂度为 $O(\sqrt{p^e}+\log r)$ ，跟大步小步法一致，仍然非常耗时。

但是考虑到 $g'$ 的阶是素数的幂的情况，其实整个算法还可以从 $O(\sum_{i=1}^n{ \sqrt{p_i^{e_i}}+\log r})$ 优化到 $O(\sum_{i=1}^n{e_i \sqrt{p_i}+\log r})$。

#### 优化

Main observation 是新的离散对数问题的元素的阶是 $p_i^{e_i}$，也就是一个质数的幂。对于这种情况，有一个算法可以把求解离散对数问题的复杂度从 $O(\sqrt{p_i^{e_i}})$优化到 $O(e^i\sqrt{p_i})$。

WLOG，给定 $g, h=g^x$ ，且 $g$ 的阶为 $p^e$。我们首先把 $x$ 写作 $x=x_0 + x_1p+ x_2p^2 + \dots + x_{e-1}p^{e-1}$。考虑 $h^{ p^{e-1} }$:

$$
\begin{aligned}
h^{ q^{e-1} }&=( g^x )^{ p^{e-1} }\\
&=g^{(x_0 + x_1p+ x_2p^2 + \dots + x_{e-1}p^{e-1})p^{e-1}}\\
&=g^{ x_0p^{e-1}}\\
&=(g^{ p^{e-1}})^{ x_0}
\end{aligned}
$$

等式左边 $h^{ q^{e-1} }$ 和右边 $g^{ p^{e-1}}$ 都可以求出来，于是离散对数问题规约到一个更小的问题上，此时 $g^{ p^{e-1}}$ 的阶是 $p$ ，那么我们可以利用小步大步法在 $O(\sqrt{p})$ 时间内计算得到 $x_0$。同理根据 $h^{e-2}$ 求 $x_1$，以此类推。

于是这个离散对数问题的求解被优化到 $O(e\sqrt{p})$，整个 Pohlig-Hellman 的时间复杂度被优化到 $O(\sum_{i=1}^n{e_i \sqrt{p_i}+\log r})$。

SageMath 为我们封装了以上两种求离散对数的函数：

```sage
R = GF(941)
h = R(390)
g = R(627)

# 小步大步法
x = h.log(g)  # 347
assert g**x == h


# Pohlig-Hellman 法
# 默认群运算是乘法，如果是加法，则需要指定参数 operation='+'
# 还可以指定生成元的阶 order=xx，可以优化效率
x = discrete_log(h, g)  # 347
assert g**x == h
```

## 3. Solution

从上述背景知识可以知道，对于一个群阶为 $n$ 的阿贝尔群，如果群阶不能被分解成若干个小素数的幂的乘积形式，那么这个阿贝尔群的离散对数问题是困难的，因为只能使用小步大步法计算；如果群阶可以被分解成若干小素数的幂的成绩形式，那么这个阿贝尔群的离散对数问题可以使用 Pohlig-Hellman 算法有效求出。

题目给定的椭圆曲线是 BLS12-381 ，它的安全参数信息在[这里](https://neuromancer.sk/std/bls/BLS12-381)。

初步分析，如果要求出陷门信息 $s$ ，第一反应就是求解离散对数问题。那么首先看 Trusted Setup 是否使用安全的循环子群，如果正确使用，那么应该满足 $( g_1^s )^r=1$ .

使用 SageMath：

```sage
# 1. 检查是否在安全循环子群中
st0 = E(0x0F99F411A5F6C484EC5CAD7B9F9C0F01A3D2BB73759BB95567F1FE4910331D32B95ED87E36681230273C9A6677BE3A69, 0x12978C5E13A226B039CE22A0F4961D329747F0B78350988DAB4C1263455C826418A667CA97AC55576228FC7AA77D33E5)
# expected: (0: 1: 0)
print(st0 * 0x73EDA753299D7D483339D80809A1D80553BDA402FFFE5BFEFFFFFFFF00000001)
```

输出结果不是无穷远点，所以题目没有使用安全的循环子群。打印实际的子群阶及其分解因子：

```sage
# 2. 打印实际 order并 factor
order = st0.order()
print(order)
print(factor(order))
```
结果:

```
3 * 11 * 10177 * 859267 * 52437899 * 52435875175126190479447740508185965837690552500527637822603658699938581184513
```

发现实际子群 $G_1$ 的阶是由 cofactor 中的因子和大素数因子一起组成的。为了满足 pairing， $G_2$ 的阶与 $G_1$ 的阶必须保持一致。

题目设计者如此操作之后，看似这两个拼凑出来的循环子群的阶仍然是大素数，但不再安全。攻击者可以轻易地把这些点映射到不安全的小素数子群中。记拼凑出来的循环子群 $G_1$的生成元为 $g_1$，那么 $g_1^{r_1}$ 这个元素所派生出来的循环子群的阶就变为 $3 * 11 * 10177 * 859267 * 52437899$。

在这个新的循环子群里，由于群阶没有大素数，我们可以使用 Pohlig-Hellman 算法高效的求出陷门 $s$ 模各个因子的值，进而使用 CRT 恢复出 $s$。

使用 $g_1$ 来恢复 $s\pmod{3 * 11 * 10177 * 859267 * 52437899}$：

```python
s_n1 = discrete_log(st11 * 52435875175126190479447740508185965837690552500527637822603658699938581184513, st10 * 52435875175126190479447740508185965837690552500527637822603658699938581184513, operation = '+')
# output: 2335387132884273659
```

处理 $g_2$ 时需要特别注意：由于这个二次扩域的大小几乎是 $g_1$ 阶的平方，因此这个群阶我们没办法直接通过 `order()` 方法计算。但是我们知道 $G_2$ 的阶必须与 $G_1$ 保持一致，这是一个重要信息。 $G_2$ 的阶无非也是 cofactor 中的几个因子拼凑，我们逐个尝试即可。最终尝试得到阶为 `13 * 23 * 2713 * 11953 * 262069 * 402096035359507321594726366720466575392706800671181159425656785868777272553337714697862511267018014931937703598282857976535744623203249 * 52435875175126190479447740508185965837690552500527637822603658699938581184513`. 使用同样的技巧消除两个大质因子：

```python
s_n2 = discrete_log(st21 * 402096035359507321594726366720466575392706800671181159425656785868777272553337714697862511267018014931937703598282857976535744623203249 * 52435875175126190479447740508185965837690552500527637822603658699938581184513 * 13 * 23, st20 * 402096035359507321594726366720466575392706800671181159425656785868777272553337714697862511267018014931937703598282857976535744623203249 * 52435875175126190479447740508185965837690552500527637822603658699938581184513 * 13 * 23, 13 * 23 * 2713 * 11953 * 262069, operation = '+')
# output: 6942769366567
```

再使用一遍 crt 结合起来尝试恢复 s：

```python
s_tag = crt([s_n1, s_n2], [3 * 11 * 10177 * 859267 * 52437899, 2713 * 11953 * 262069])
print("s_tag: ", s_tag)
print(len(s_tag.bits()))
# s_tag:  62308043734996521086909071585406
# 106
```

结果还不是 128 bit，最多还差 $2^{22}$ 次方。这就只能通过爆破地方式尝试补齐了，最终的结果是：`114939083266787167213538091034071020048`。

Done.






