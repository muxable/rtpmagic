#ifndef GSTREAMER_H
#define GSTREAMER_H

#include <glib.h>
#include <gst/gst.h>
#include <stdint.h>
#include <stdlib.h>

extern void goHandleAppSinkBuffer(void *, int, int, void *);
extern void goHandleRtpAppSinkBuffer(void *, int, int, void *);
extern void goHandleTwccStats(unsigned int, unsigned int, unsigned int, unsigned int, long, void *);

void gstreamer_init(void);
void gstreamer_main_loop(void);
GstElement *gstreamer_start(char *, void *);
void gstreamer_stop(GstElement *);
void gstreamer_push_rtp(GstElement *, void *, int);
void gstreamer_push_rtcp(GstElement *, void *, int);
void gstreamer_set_bitrate(GstElement *, unsigned int);

#endif