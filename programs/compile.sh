#!/bin/bash

for PROGRAM in *.c; do
	gcc $PROGRAM -o ${PROGRAM%%.*}
done
