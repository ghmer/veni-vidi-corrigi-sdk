# veni-vidi-corrigi-sdk

Image-processing building blocks for LLM-assisted correction of handwritten
class tests: a pure-Go, in-memory SDK (`imageutil`) for inspecting, zooming,
straightening and enhancing scans of handwritten class tests.

## SDK

The `github.com/ghmer/veni-vidi-corrigi-sdk/imageutil` package exposes a
`Service` with one method per operation. Every method takes a
`context.Context` and one `Input` carrying the raw JPEG/PNG bytes (or their
base64 encoding) and returns a NEW transformed image inside `Result.Data`.
The SDK never touches the file system — no workspace directory, no temp
files, no symlink or path risks; the host owns all storage and feeds results
back as inputs for follow-up calls. Regions use normalized coordinates
(0.0–1.0) relative to the original image so callers can address areas based
on the downscaled `View` image an LLM actually perceives.

Operations:

| Domain      | Methods                                                                                                                                                                   |
| ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Inspection  | `Info`, `View`                                                                                                                                                            |
| Geometry    | `Crop`, `Scale`, `Rotate`, `Split`, `TrimBorders`                                                                                                                         |
| Enhancement | `Grayscale`, `AutoContrast`, `AdjustContrast`, `AdjustBrightness`, `Gamma`, `Threshold`, `Sharpen`, `Denoise`, `Invert`                                                   |
| Annotation  | `AnnotateRect`                                                                                                                                                            |
| Pipeline    | `EnhanceForReading`                                                                                                                                                       |

## Usage

```sh
go get github.com/ghmer/veni-vidi-corrigi-sdk/imageutil
```

```go
import "github.com/ghmer/veni-vidi-corrigi-sdk/imageutil"

svc, err := imageutil.New() // defaults: budgets, max side, JPEG quality
...
res, err := svc.View(ctx, imageutil.Input{Data: rawBytes}, 1024)
...
res2, err := svc.Crop(ctx, imageutil.Input{Data: res.Data}, region, imageutil.Options{})
```

Every method validates its arguments, clamps them into safe ranges, honours
EXIF orientation, enforces byte/pixel budgets and re-encodes the result;
human-readable error messages make invalid input diagnosable.

## Development

```sh
make test    # go test -timeout=180s ./...
make lint    # golangci-lint run (quality gate)
make fmt     # gofmt -l -w .
make tidy    # go mod tidy
```

### Layout

```text
imageutil/      the image SDK public API
internal/       SDK internals (codec, geometry, enhance, exif, ...)
```
