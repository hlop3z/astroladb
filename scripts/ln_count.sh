#!/bin/bash

tokei . -t Go --exclude '*_test.go'
# tokei . -t Go --sort code
