---
layout: post
title: 以太坊账户生成
tags: [ethereum]
---

# 以太坊新账户生成
以太坊的生成新账号的流程。

## 生成ECDSA Key
1. 通过`privateKeyECDSA, err := ecdsa.GenerateKey(crypto.S256(), rand)`生成一对新的ecdsa key pair；
使用secp256k1算法。
2. 通过publicKey生成address，`common.BytesToAddress(Keccak256(pubBytes[1:])[12:])`；pubBytes为公钥字节
数组。
3. 使用password对秘钥进行加密，之后生成json文件并将其进行保存。

注：使用geth account new生成的账号并不是hd钱包，以太坊的hd钱包派生路径为`m/44'/60'/0'/1`