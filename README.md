# wiregenx

`wiregenx` — Google Wire uchun oddiy generator.
Entry point’lar (http / grpc / worker) uchun `wire.go` fayllarni avtomatik yaratadi.

---

## Installation

```bash
go install github.com/your-org/wiregenx@latest
```

yoki loyihaga dependency sifatida:

```bash
go get github.com/muhriddinnorqulov/wiregenx
```

---

## Usage

Wire fayllarni generatsiya qilish:

```bash
wiregenx --root ./path/to/your/project --out ./path/to/output/directory
```

## Requirements

* Go 1.20+
* `github.com/google/wire`

---

