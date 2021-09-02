## What?

Visit analyzes the actions of committees and attestations. It visits the eth2
beacon chain - without participating - and observes it. Its current job is to
figure out which validators are slow or missing and visualize their patterns.

It talks to an eth2 node using the Eth2 Beacon Node API.

## How?

You can easily use visit to collect data over 10 epochs, index them and
visualize them. The visit command below is gonna data collect for about an hour
(10 epochs).

If you want to tweak the number of epochs or any other parameter, you will have
to change the code!

```
$ go build
$ ./visit 127.0.0.1:4000 # point it to your beacon API port
$ cd visualize_d3
$ python3 retrieve_data_from_db.py
$ firefox index.html
```
