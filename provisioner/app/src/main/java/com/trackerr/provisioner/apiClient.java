package com.trackerr.provisioner;

import android.content.Context;

import com.android.volley.DefaultRetryPolicy;
import com.android.volley.Request;
import com.android.volley.RequestQueue;
import com.android.volley.toolbox.JsonArrayRequest;
import com.android.volley.toolbox.JsonObjectRequest;
import com.android.volley.toolbox.Volley;
import com.trackerr.provisioner.models.JsonResponse;

import org.json.JSONArray;
import org.json.JSONException;
import org.json.JSONObject;

import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ExecutionException;

public class apiClient {
    private String baseurl = "https://example.com:8080/api/v1/";
    private RequestQueue requestQueue;
    private Map<String, String> headers;

    public apiClient(Context context, String key){
        // Create request queue and add api key to header
        this.requestQueue = Volley.newRequestQueue(context.getApplicationContext());
        this.headers = new HashMap<>();
        this.headers.put("X-API-Key", key);
    }

    // Send GET request to models endpoint
    public CompletableFuture<JsonResponse> getModels(){
        return sendJSONArrayRequest("models",null, Request.Method.GET);
    }

    // Send GET request to whoami endpoint
    public CompletableFuture<String> whoami(){
        return sendJSONObjectRequest("whoami",null, Request.Method.GET).thenApply(result -> {
            try {
                return result.jsonObject.getString("name");
            } catch (JSONException e) {
                return "Unknown";
            }
        });
    }

    // Poll the api every second until tracker is connected
    public CompletableFuture<Void> waitUntilTrackerConnected(String trackerId){
        CompletableFuture<Void> future = new CompletableFuture<>();
        new Thread(() -> {
            try {
                while(true) {
                    JSONObject tracker = getTrackerInfo(trackerId).get().jsonObject;
                    if (tracker == null){
                        future.completeExceptionally(new Exception("Failed: Tracker with the same id or name is already registered with another user. "));
                        return;
                    }
                    boolean isConnected = tracker.getBoolean("Connected");
                    // Complete future if tracker has connected
                    if (isConnected) {
                        future.complete(null);
                        break;
                    }
                    Thread.sleep(1000);
                }
            } catch (ExecutionException | InterruptedException | JSONException e) {
                throw new RuntimeException(e);
            }
        }).start();
        return future;
    }

    // Send GET request to endpoint for specific tracker
    public CompletableFuture<JsonResponse> getTrackerInfo(String trackerId) {
        return sendJSONObjectRequest("trackers/"+trackerId,null,Request.Method.GET);
    }

    // Send POST request with tracker data to trackers endpoint to register tracker
    public CompletableFuture<JsonResponse> registerTracker(TrackerDetails trackerDetails) throws JSONException {
        JSONObject data = trackerDetails.toJsonObject();
        return sendJSONObjectRequest("trackers",data,Request.Method.POST);
    }

    // Send HTTP request to API with nullable JSON object body
    private CompletableFuture<JsonResponse> sendJSONObjectRequest(String resource,
                                                                  JSONObject requestBody,
                                                                  int method
                                                      ) {
        CompletableFuture<JsonResponse> future = new CompletableFuture<>();
        String url = baseurl + resource;
        JsonObjectRequest jsonRequest = new JsonObjectRequest(method,
                url,
                requestBody,
                response -> future.complete(new JsonResponse(response,200)),
                error -> {
                    if (error.networkResponse == null){
                        future.completeExceptionally(error);
                    } else {
                        future.complete(new JsonResponse(error.networkResponse.statusCode));
                    }
                }) {
            @Override
            public Map<String, String> getHeaders() {
                return headers;
            }
        };
        jsonRequest.setRetryPolicy(new DefaultRetryPolicy(
                2000,
                2,
                1));
        requestQueue.add(jsonRequest);
        return future;
    }

    // Send HTTP request to API with nullable JSON array body
    private CompletableFuture<JsonResponse> sendJSONArrayRequest(String resource,
                                                          JSONArray requestBody,
                                                          int method){
        CompletableFuture<JsonResponse> future = new CompletableFuture<>();
        String url = baseurl + resource;
        JsonArrayRequest jsonRequest = new JsonArrayRequest(method,
                url,
                requestBody,
                response -> future.complete(new JsonResponse(response,200)),
                error -> {
                    if (error.networkResponse == null){
                        future.completeExceptionally(error);
                    } else {
                        future.complete(new JsonResponse(error.networkResponse.statusCode));
                    }
                }){
            @Override
            public Map<String, String> getHeaders() {return headers;}
        };
        jsonRequest.setRetryPolicy(new DefaultRetryPolicy(
                2000,
                2,
                1));
        requestQueue.add(jsonRequest);
        return future;
    }
}


