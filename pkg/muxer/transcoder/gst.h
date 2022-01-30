#ifndef GST_H
#define GST_H

#include <glib.h>
#include <gst/gst.h>
#include <stdint.h>
#include <stdlib.h>

extern void goHandleVideoPipelineBuffer(void *buffer, int bufferLen, uint64_t samples, uint64_t pts, void *data);
extern void goHandleVideoPipelineRtp(void *buffer, int bufferLen, uint64_t samples, void *data);
extern void goHandleAudioPipelineBuffer(void *buffer, int bufferLen, uint64_t samples, uint64_t pts, void *data);

GstElement *gstreamer_send_create_pipeline(char *pipeline);
void gstreamer_send_start_pipeline(GstElement *pipeline, void *data);
void gstreamer_send_stop_pipeline(GstElement *pipeline);
void gstreamer_set_video_bitrate(GstElement *, unsigned int);
void gstreamer_set_packet_loss_percentage(GstElement *, unsigned int);

#endif
