README
======

App de budgeting.

## MongoDB

### Data folder

To see where your data is being saved on the host's system:

```bash
docker volume inspect budget_mongo-data
```

This will show you a JSON object, and you'll see a path under the `Mountpoint` property.

In my case, it's pointing to `/var/lib/docker/volumes/budget_mongo-data/_data`

## Environment variables

- `GOOGLE_OAUTH2_CLIENT_ID`: required
- `GOOGLE_OAUTH2_CLIENT_SECRET`: required
- `PORT`: optional (defaults to `8080`)

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

