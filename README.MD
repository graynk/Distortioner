# Distortioner

Telegram bot for distorting pictures, stickers and GIFs using Content Aware Scale.
Currently resides here: [@distortionerbot](https://t.me/distortionerbot)

## Docker support
Fill out your bot token in distortioner.env, then launch as usual:
```Bash
sudo docker run -d --restart unless-stopped \
   --name distortioner \
   --env-file distortioner.env \
   ghcr.io/graynk/distortioner:latest
```

With Podman:
```Bash
podman run -d \
   --label "io.containers.autoupdate=image" \
   --name distortioner \
   --env-file distortioner.env \
   ghcr.io/graynk/distortioner:latest
cd ~/.config/systemd/user/
podman generate systemd --new --files --name distortioner
systemctl enable --user container-distortioner.service
```

## TODO
* Distort video notes (with sound)
* Distort text (why not)
* Distort voice messages (since I will have to do that anyway for video notes)
* Make distortion increase throughout the video
