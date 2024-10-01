---
title: Crypto - Let's hash it out
description: 2023 | ZKHACK | Crypto
---

- [1. Puzzle Description](#1-puzzle-description)
- [2. Preliminary](#2-preliminary)
- [3. Solution](#3-solution)
- [4. Code](#4-code)

## 1. Puzzle Description

It all feels random, but it might not be.

> https://zkhack.dev/events/puzzle1.html

## 2. Preliminary

1. BLS 签名方案：基于 paring，分为 3 个算法：
   
   给定素数阶 $r$ 循环群 $\mathbb{G}$ 的生成元 $g$ ，双线性配对 $e$ 。(公开的)

   - KeyGen: 生成有限域 $GF(r)$ 中的随机标量 $sk$ 作为私钥，然后生成公钥 $pk = sk \cdot g$ .
   - Sign: 给定消息 $m\in\mathbb{G}$ ，那么该消息的签名为： $\Sigma=sk\cdot m$ .
   - Verify: 给定 $m, \Sigma, pk$ ，验证等式： $e(m, pk)= e(\Sigma, g)$

   注意， $m, \Sigma$ 是 $G_1$ 元素， $pk,g$ 是 $G_2$ 元素。

2. 一般而言，消息都是任意长度的字符串，也可以理解为字节数组，或者一个比特流。为了使用 BLS 签名方案，则需要进行两步操作：一是把任意长度的字符串映射到固定长度的字符串，二是把固定长度字符串映射到群元素。

   如上所述，Challenge采用的方案是首先用 blake2b hash 算法把消息转换为 256 bits ，然后使用 `hash_to_curve` 技术把 256bits 串映射到群元素（Pedersen hash）。

   - Blake2b Hash (`Cargo.toml` 添加 `blake2s_simd="version"`  )
        
        ```Rust
        fn main() {
            let msg = "I love you";
            let msg_bytes = msg.as_bytes();
            let h = blake2s_simd::blake2s(msg_bytes);
            println!("{:?}", h);
        }
        // Hash(0x7b9db068deeda65efb3f07a3bab9242ad4e5fdbf3bf3f4d229c82f2ebc4beb90)
        ```

   - Pedersen Hash (`Cargo.toml` 添加 G1 element type of pairing-friendly curve, )

     inputs: 固定长度 bits 串; output: G1 element

     我们首先把 inputs 按照 $r$ 大小分成 $k$ 组， 即 $|bits|=r * k$ 。然后随机在指定配对友好型椭圆曲线的素数阶子群中选择 $k$ 个生成元 $g_1, g_2, \dots, g_k$ 。这些生成元必须是随机均匀取样，使得两两之间的关系是无从知晓的。最后计算 pedersen hash $h=m_1g_1+m_2g_2+\dots+m_kg_k$ 。其中如何把 $r$ 大小的 bits 分组编码成 $m_i$ 需要设计成一个 encoding_function （可以简单地将其转化成 ASCII 码，也可以自定义，比如Zcash）。

     因此，为了生成 pedersen hash 我们需要引入 pairing-friendly curve 、安全随机源、已经实现好的 pedersen primitive。

     ```Rust
     use ark_bls12_381::{G1Affine, G1Projective};
     use ark_crypto_primitives::crh::{
         pedersen::{Window, CRH},
         CRH as CRHScheme
     };
     use blake2s_simd::Hash;
     use rand::SeedableRng;
     use rand_chacha::ChaCha20Rng;
     
     // Why?
     #[derive(Clone)]
     struct PedersenWindow {}
     
     impl Window for PedersenWindow {
         const WINDOW_SIZE: usize = 1;
         const NUM_WINDOWS: usize = 256;
     }
     
     pub fn hash_to_curve(msg: &[u8]) -> (Vec<u8>, G1Affine) {
         let rng_pedersen = &mut ChaCha20Rng::from_seed([
             1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
             1, 1,
         ]);
         let parameters = CRH::<G1Projective, PedersenWindow>::setup(rng_pedersen).unwrap();
         let b2hash = blake2s_simd::blake2s(msg);
         (
             b2hash.as_bytes().to_vec(),
             CRH::<G1Projective, PedersenWindow>::evaluate(&parameters, b2hash.as_bytes()).unwrap()
         )
     }
     
     fn main() {
         let msg = "I love you";
         let msg_bytes = msg.as_bytes();
         let (b2hash, phash) = hash_to_curve(msg_bytes);
         println!("{:?}, {}", &b2hash, &phash);
     
         let a: &[u8; 32] = b2hash.as_slice().try_into().unwrap();
         let aa = Hash::from(a);
         println!("{:?}", &aa);
     }
     // [123, 157, 176, 104, 222, 237, 166, 94, 251, 63, 7, 163, 186, 185, 36, 42, 212, 229, 253, 191, 59, 243, 244, 210, 41, 200, 47, 46, 188, 75, 235, 144], GroupAffine(x=Fp384 "(159228A979000DA250128CABDE97477EEC6A0CADFAE580B7D8AD46F185F4B43AC373CB877FFFEFAB0172AF25562DAF4A)", y=Fp384 "(04CEBA318A27291276178BDD9093432691D6E9D8E7178CC75FAB366225D332012288DF8EB3CA48D828821EA37D78BC3A)")
     // Hash(0x7b9db068deeda65efb3f07a3bab9242ad4e5fdbf3bf3f4d229c82f2ebc4beb90)
     ```

## 3. Solution

注意到 `hash_to_curve` 函数使用了同样的伪随机数生成器来挑选群生成元 $g_1, g_2, \dots, g_{256}$ ，这意味着每个消息的签名的计算方式为： $\Sigma_i=sk\cdot[b_1(m_i) g_1+b_2(m_i) g_2+\dots+b_{256}(m_i) g_{256}]$ ，其中 $m_i$ 表示第 $i$ 个消息， $b(m_i)$ 表示对第 $i$ 个消息进行 blake2s hash的哈希值，$b_j(m_i)$ 表示哈希值的第 $j$ 位，中括号（[]）里面的内容就是每个 `blake2s hash` 的 `pedersen hash` 结果，最后使用 $sk$ 进行签名。

从上述等式可知：由于 $g_1, g_2, \dots, g_{256}$ 和 $sk$ 均固定，那么任意两个签名之间是可以进行加法运算的，然后得到另一个未知消息的签名。解了这个【线性特性】之后，我们就可以解决这个puzzle了。

题目要我们给我们自定义的名字生成相应的签名，我们首先可直接算出名字的 `blake2s hash`  ，当然我们可以继续计算 `pedersen hash` ，但是由于我们没有 $sk$ 信息，因此没办法直接计算出目标签名。

但是注意到，我们拥有 256 个消息及其签名，这意味我们可以进行线性组合（Linear Combination）。诚然，两个签名相加可以得到一个未知消息的签名，这个消息是不可控的，甚至是无意义的（因为是与生成器相乘的标量是布尔值，如果两个布尔值都为 1 ，那么加起来的结果为 2，显然不可能有某一个 hash 的二进制表示中的某一位是 2）。但是，线性代数教给我们一个利器：线性组合。我们拥有256个消息，那么其 `blake2s hash` 的每一位我们都是知道的，因此把这些消息的二进制表示作为行向量，形成了一个 $256*256$ 的矩阵 $A$ ，我们的目标是生成一个 $1 * 256$ 行向量 $y$ 。那么问题就转换为，寻找一个线性组合解 $x$ ，满足 $xA=y$ 。有了这个线性组合解，我们就可以使用这 256 个消息来构造出我们想要的消息 $m$ ，又因为签名具有线性特性，那么使用这个解去线性组合这 256 个签名得到的就是目标消息的签名。

需要注意的是，由于我们最终需要使用解去线性组合签名，而后续与生成器进行乘法操作时要求标量在 $F_r$ 上，因此在进行矩阵运算时，我们需要把所有的元素都表示在 $F_r$ 中。

数学表达为：

$$
\vec{x}\cdot sk\cdot \begin{pmatrix} b_1(m_1)&b_2(m_1)&\dots&b_{256}(m_1)\\ b_1(m_2)&b_2(m_2)&\dots&b_{256}(m_2)\\ \vdots&\vdots&\ddots&\vdots\\ b_1(m_{256})&b_2(m_{256})&\dots&b_{256}(m_{256 })\\ \end{pmatrix} \cdot \begin{pmatrix} g_1\\g_2\\\vdots\\g_{256} \end{pmatrix} =\vec{x}\cdot \begin{pmatrix} \Sigma_1\\\Sigma_2\\\vdots\\\Sigma_{256} \end{pmatrix}
$$

## 4. Code

见 [code](https://github.com/Xor0v0/ZK-Puzzle-Writeups/tree/main/docs/crypto/lets-hash-it-out/code) 文件夹。