# PromNATS Docker Proxy

* Gets information about posible metrics to scrape via docker labels.
* Builds an internal directory to support service discoverty
* Upon request via NATS do a GET, via http, metrics from matching container(s).
