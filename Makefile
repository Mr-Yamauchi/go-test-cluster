SUBDIRS = ./controller ./rmanager

all : $(SUBDIRS) 

clean : $(SUBDIRS)

$(SUBDIRS) : FORCE
	make -C $@ $(MAKECMDGOALS)

FORCE:
