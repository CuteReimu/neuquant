# NeuQuant 图像量化算法
![](https://img.shields.io/github/languages/top/CuteReimu/neuquant "语言")
[![](https://img.shields.io/github/workflow/status/CuteReimu/neuquant/Go)](https://github.com/CuteReimu/neuquant/actions/workflows/golangci-lint.yml "代码分析")

Neural network based color quantizer. Can be used to transform image.Image to image.Paletted.

颜色量化的神经网络算法。可以用来把任意图像转化为image.Paletted图像。

## Install 安装方法

```
go get github.com/CuteReimu/neuquant
```

## Usage 使用方法

```go
package main

import (
	"github.com/CuteReimu/neuquant"
	"image"
	"image/gif"
	"image/png"
	"os"
)

func main() {
	f, _ := os.Open("1.png")
	defer f.Close()
	img, _ := png.Decode(f)

	img2 := neuquant.Paletted(img)
	
	f2, _ := os.Create("1.gif")
	defer f2.Close()
	_ = gif.Encode(f2, img2, nil)
}

```

## License

The original NeuQuant Algorithm was developed by Anthony Dekker, 1994. See 'LICENSE'.

Golang implementation of NeuQuant Algorithm was done by CuteReimu, 2021.
