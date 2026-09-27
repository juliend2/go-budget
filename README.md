README
======

App de budgeting.

## Installation

```
make install
```

Installs the required apt packages and a Go toolchain matching the version in
`go.mod`, then downloads the Go modules.

## Environment variables

- `GOOGLE_OAUTH2_CLIENT_ID`: required
- `GOOGLE_OAUTH2_CLIENT_SECRET`: required
- `ALLOWED_EMAILS`: required — comma-separated list of Google account emails
  allowed to log in (e.g. `alice@gmail.com,bob@gmail.com`). Comparison is
  case-insensitive. The app refuses to start if this is empty.
- `MONGO_URI`: optional (defaults to `mongodb://root:password@localhost:27017/admin`).
  `make run` overrides it to `mongodb://localhost:27027` (the local dev
  instance, no auth); pass your own to point somewhere else.
- `PORT`: optional (defaults to `8080`)

## Local development

- `make mongo-local-install`, `make mongo-local`, `make mongo-local-stop`:
  MongoDB 4.4 as a tarball under `~/mongodb-budget` (no root needed). 
- `make run`: start the app (uses `MONGO_URI`, defaults to the local instance
  on `localhost:27027`).
- `make test`: run the tests.

## Backup & restore

Dump production through an SSH tunnel (after `make mongo-local-install`, the
tools live in `~/mongodb-budget/bin`). The production host is deliberately not
committed to this repo: it lives in `.env.deploy` (git-ignored) as
`DEPLOY_HOST=user@your-server`, which `make deploy` also reads:

```bash
ssh -f -N -L 27018:localhost:27017 "$DEPLOY_HOST"
~/mongodb-budget/bin/mongodump \
  --uri="mongodb://root:password@localhost:27018/budget?authSource=admin" \
  --gzip --out="$HOME/backups/budget-mongo-$(date +%F)"
pkill -f "27018:localhost:27017"
```

Restore a dump (locally or to any instance):

```bash
~/mongodb-budget/bin/mongorestore --uri="mongodb://localhost:27027" --drop --gzip <backup-dir>
```

## Deployment

1. create the `/etc/systemd/system/budget.service` file, by copying and editing the `budget.service` in
   this repo.
2. then do:
    ```bash
    chmod +x [...DIR WHERE THE EXECUTABLE IS LOCATED...]
    # or its parent dir
    sudo chown -R www-data:www-data [...DIR WHERE THE EXECUTABLE IS LOCATED...]
    sudo systemctl daemon-reload
    sudo systemctl enable --now budget
    ```
3. verify it's running:
    ```bash
    sudo systemctl status budget
    ```

## TODO

- [x] `ExpenseTemplate`: CRUD
- [ ] `Payment`
    - [ ] UI moins complexe pour l'ajout des `Payment`
    - [ ] Delete action (besoin rare mais necessaire parfois)

