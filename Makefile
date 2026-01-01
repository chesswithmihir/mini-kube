# Root Makefile for Mini-Kube

.PHONY: all my-runc my-kube clean test

all: my-runc my-kube

my-runc:
	$(MAKE) -C my-runc build

my-kube:
	$(MAKE) -C my-kube build

test:
	$(MAKE) -C my-runc test
	$(MAKE) -C my-kube test

clean:
	$(MAKE) -C my-runc clean
	$(MAKE) -C my-kube clean
