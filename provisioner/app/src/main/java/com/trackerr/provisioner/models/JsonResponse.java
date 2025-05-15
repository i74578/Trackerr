package com.trackerr.provisioner.models;

import org.json.JSONArray;
import org.json.JSONObject;

public class JsonResponse {
    public final JSONObject jsonObject;
    public final JSONArray jsonArray;
    public final int statusCode;
    public JsonResponse(JSONObject jsonObject, JSONArray jsonArray, int statusCode){
        this.jsonArray = jsonArray;
        this.jsonObject = jsonObject;
        this.statusCode = statusCode;
    }

    public JsonResponse(JSONArray jsonArray, int statusCode){
        this(null,jsonArray,statusCode);
    }
    public JsonResponse(JSONObject jsonObject, int statusCode){
        this(jsonObject,null,statusCode);
    }
    public JsonResponse(int statusCode){
        this(null,null,statusCode);
    }
}