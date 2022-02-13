#ifndef ENCODER_H
#define ENCODER_H

#include <glib.h>
#include <gst/gst.h>

extern gboolean goBusFunc(GstBus *, GstMessage *, gpointer);

gboolean cgoBusFunc(GstBus *, GstMessage *, gpointer);

#endif