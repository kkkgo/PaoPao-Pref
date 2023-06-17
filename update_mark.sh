#!/bin/sh
docker run -d --name mark_builder sliamb:/mark_builder
rm global_mark.dat global_mark.dat.sha256sum
docker cp mark_builder:/pub/global_mark.dat .
docker cp mark_builder:/pub/global_mark.dat.sha256sum .