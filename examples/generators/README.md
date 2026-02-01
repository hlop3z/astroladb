# README

## Run

```sh
alab gen run generators/fastapi.js  -o ./generated
```

## Start App

```sh
uv init
```

## Install Dependencies

```sh
uv add fastapi[standard] pydantic && uv add --dev ruff mypy
```

## Linting (Ruff)

```sh
uv run ruff check auth
```

## Linting (MyPy)

```sh
uv run mypy auth/
```

## Run App

```sh
uv run fastapi dev main.py
```
