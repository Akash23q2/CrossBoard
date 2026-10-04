## Allow firewall permissions
```
sudo ufw allow from 10.0.0.0/8 to any port 5050 proto tcp
sudo ufw allow from 172.16.0.0/12 to any port 5050 proto tcp
sudo ufw allow from 192.168.0.0/16 to any port 5050 proto tcp
```