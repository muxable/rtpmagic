#ifndef GSTREAMER_H
#define GSTREAMER_H

#include <glib.h>
#include <gst/gst.h>
#include <stdint.h>
#include <stdlib.h>

extern void goHandleRtpAppSinkBuffer(void *, int, int, void *);

GstElement *gstreamer_start(char *, void *);
void gstreamer_stop(GstElement *);

#endif