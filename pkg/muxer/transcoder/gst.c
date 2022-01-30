#include "gst.h"

#include <gst/app/gstappsrc.h>
#include <stdio.h>

static gboolean gstreamer_send_bus_call(GstBus *bus, GstMessage *msg, gpointer data)
{
  switch (GST_MESSAGE_TYPE(msg))
  {

  case GST_MESSAGE_EOS:
    g_print("End of stream\n");
    exit(1);
    break;

  case GST_MESSAGE_ERROR:
  {
    gchar *debug;
    GError *error;

    gst_message_parse_error(msg, &error, &debug);
    g_free(debug);

    g_printerr("Error: %s\n", error->message);
    g_error_free(error);
    exit(1);
  }
  default:
    break;
  }

  return TRUE;
}

GstFlowReturn gstreamer_send_new_video_sample_handler(GstElement *object, gpointer user_data)
{
  GstSample *sample = NULL;
  GstBuffer *buffer = NULL;
  gpointer copy = NULL;
  gsize copy_size = 0;

  g_signal_emit_by_name(object, "pull-sample", &sample);
  if (sample)
  {
    buffer = gst_sample_get_buffer(sample);
    if (buffer)
    {
      gst_buffer_extract_dup(buffer, 0, gst_buffer_get_size(buffer), &copy, &copy_size);
      goHandleVideoPipelineBuffer(copy, copy_size, GST_BUFFER_DURATION(buffer), GST_BUFFER_DTS(buffer), user_data);
    }
    gst_sample_unref(sample);
  }

  return GST_FLOW_OK;
}

GstFlowReturn gstreamer_send_new_video_rtp_handler(GstElement *object, gpointer user_data)
{
  GstSample *sample = NULL;
  GstBuffer *buffer = NULL;
  gpointer copy = NULL;
  gsize copy_size = 0;

  g_signal_emit_by_name(object, "pull-sample", &sample);
  if (sample)
  {
    buffer = gst_sample_get_buffer(sample);
    if (buffer)
    {
      gst_buffer_extract_dup(buffer, 0, gst_buffer_get_size(buffer), &copy, &copy_size);
      goHandleVideoPipelineRtp(copy, copy_size, GST_BUFFER_DURATION(buffer), user_data);
    }
    gst_sample_unref(sample);
  }

  return GST_FLOW_OK;
}

GstFlowReturn gstreamer_send_new_audio_sample_handler(GstElement *object, gpointer user_data)
{
  GstSample *sample = NULL;
  GstBuffer *buffer = NULL;
  gpointer copy = NULL;
  gsize copy_size = 0;

  g_signal_emit_by_name(object, "pull-sample", &sample);
  if (sample)
  {
    buffer = gst_sample_get_buffer(sample);
    if (buffer)
    {
      gst_buffer_extract_dup(buffer, 0, gst_buffer_get_size(buffer), &copy, &copy_size);
      goHandleAudioPipelineBuffer(copy, copy_size, GST_BUFFER_DURATION(buffer), GST_BUFFER_DTS(buffer), user_data);
    }
    gst_sample_unref(sample);
  }

  return GST_FLOW_OK;
}

GstElement *gstreamer_send_create_pipeline(char *pipeline)
{
  gst_init(NULL, NULL);
  GError *error = NULL;
  return gst_parse_launch(pipeline, &error);
}

void gstreamer_send_start_pipeline(GstElement *pipeline, void *data)
{
  GstBus *bus = gst_pipeline_get_bus(GST_PIPELINE(pipeline));
  gst_bus_add_watch(bus, gstreamer_send_bus_call, NULL);
  gst_object_unref(bus);

  GstElement *videoappsink = gst_bin_get_by_name(GST_BIN(pipeline), "videoappsink");
  if (videoappsink != NULL)
  {
    g_object_set(videoappsink, "emit-signals", TRUE, NULL);
    g_signal_connect(videoappsink, "new-sample", G_CALLBACK(gstreamer_send_new_video_sample_handler), data);
    gst_object_unref(videoappsink);
  }

  GstElement *videortpsink = gst_bin_get_by_name(GST_BIN(pipeline), "videortpsink");
  if (videortpsink != NULL)
  {
    g_object_set(videortpsink, "emit-signals", TRUE, NULL);
    g_signal_connect(videortpsink, "new-sample", G_CALLBACK(gstreamer_send_new_video_rtp_handler), data);
    gst_object_unref(videortpsink);
  }

  GstElement *audioappsink = gst_bin_get_by_name(GST_BIN(pipeline), "audioappsink");
  g_object_set(audioappsink, "emit-signals", TRUE, NULL);
  g_signal_connect(audioappsink, "new-sample", G_CALLBACK(gstreamer_send_new_audio_sample_handler), data);
  gst_object_unref(audioappsink);

  gst_element_set_state(pipeline, GST_STATE_PLAYING);
}

void gstreamer_send_stop_pipeline(GstElement *pipeline)
{
  gst_element_set_state(pipeline, GST_STATE_NULL);
}

void gstreamer_set_video_bitrate(GstElement *pipeline, unsigned int bitrate)
{
  GstElement *encoder = gst_bin_get_by_name(GST_BIN(pipeline), "videoencode");
  if (encoder != NULL) {
    g_object_set(G_OBJECT(encoder), "bitrate", bitrate, NULL);
    gst_object_unref(encoder);
  }
}

void gstreamer_set_packet_loss_percentage(GstElement *pipeline, unsigned int plp)
{
  GstElement *encoder = gst_bin_get_by_name(GST_BIN(pipeline), "audioencode");
  g_object_set(G_OBJECT(encoder), "packet-loss-percentage", plp, NULL);
  gst_object_unref(encoder);
}
