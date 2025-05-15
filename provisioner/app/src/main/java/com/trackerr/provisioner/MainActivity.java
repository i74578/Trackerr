package com.trackerr.provisioner;

import static com.trackerr.provisioner.Utils.blockUserInteractions;
import static com.trackerr.provisioner.Utils.showCriticalMsg;
import static com.trackerr.provisioner.Utils.showMessage;

import android.Manifest;
import android.content.Context;
import android.content.Intent;
import android.content.SharedPreferences;
import android.content.pm.PackageManager;
import android.os.Bundle;
import android.view.View;
import android.widget.ArrayAdapter;
import android.widget.Button;
import android.widget.CheckBox;
import android.widget.EditText;
import android.widget.ImageView;
import android.widget.LinearLayout;
import android.widget.Spinner;
import android.widget.TextView;

import androidx.activity.EdgeToEdge;
import androidx.annotation.NonNull;
import androidx.appcompat.app.AppCompatActivity;
import androidx.core.content.ContextCompat;
import androidx.core.graphics.Insets;
import androidx.core.view.ViewCompat;
import androidx.core.view.WindowInsetsCompat;

import com.google.gson.Gson;
import com.google.mlkit.vision.barcode.common.Barcode;
import com.google.mlkit.vision.codescanner.GmsBarcodeScanner;
import com.google.mlkit.vision.codescanner.GmsBarcodeScannerOptions;
import com.google.mlkit.vision.codescanner.GmsBarcodeScanning;
import com.trackerr.provisioner.models.JsonResponse;

import org.json.JSONArray;
import org.json.JSONException;
import org.json.JSONObject;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.CompletionException;
import java.util.concurrent.Executor;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;



public class MainActivity extends AppCompatActivity {
    HashMap<String, ModelCommands> modelCommandsMap = new HashMap<>();
    private ImageView settingsBtn;
    private EditText phonenumberField, idField, nameField;
    private Spinner modelSpinner;
    private List<String> spinnerItems;
    private ArrayAdapter<String> adapter;
    private GmsBarcodeScanner scanner;
    private LinearLayout loadingLayout, innerLayout;
    private TextView keyOwnerText, loadingText;
    private Executor executor;
    private apiClient apiClient;
    private CheckBox validateProvisioning;

    private boolean reloadModelsFlag = true;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);

        EdgeToEdge.enable(this);
        setContentView(R.layout.activity_main);

        // Reference UI components
        Button submitBtn = findViewById(R.id.submitBtn);
        ImageView scanIdBtn = findViewById(R.id.scanIdBtn);
        ImageView scanNameBtn = findViewById(R.id.scanNameBtn);
        validateProvisioning = findViewById(R.id.validateProvisioning);
        settingsBtn = findViewById(R.id.settingsBtn);
        modelSpinner = findViewById(R.id.modelSpinner);
        phonenumberField = findViewById(R.id.phonenumberField);
        idField = findViewById(R.id.idField);
        nameField = findViewById(R.id.nameField);
        loadingLayout = findViewById(R.id.loadingLayout);
        innerLayout = findViewById(R.id.innerLayout);
        loadingText = findViewById(R.id.loadingText);
        keyOwnerText = findViewById(R.id.apiKeyOwnerText);

        //Use array adapter to link the spinnerItems array to the models spinner component
        spinnerItems = new ArrayList<>();
        adapter = new ArrayAdapter<>(this, android.R.layout.simple_spinner_item, spinnerItems);
        adapter.setDropDownViewResource(android.R.layout.simple_spinner_dropdown_item);
        modelSpinner.setAdapter(adapter);

        executor = Executors.newCachedThreadPool();

        scanner = ScanBuilder();

        // Set button event listenrs
        submitBtn.setOnClickListener(v -> provision());
        scanIdBtn.setOnClickListener(v -> launchBarcodeScanner(idField));
        scanNameBtn.setOnClickListener(v -> launchBarcodeScanner(nameField));
        settingsBtn.setOnClickListener(v -> {
            // Set reload models flag, to reload models, fater returning to main screen
            reloadModelsFlag = true;
            // Return to mainactivity
            startActivity(new Intent(MainActivity.this, LoginActivity.class));
        });

        ViewCompat.setOnApplyWindowInsetsListener(findViewById(R.id.main), (v, insets) -> {
            Insets systemBars = insets.getInsets(WindowInsetsCompat.Type.systemBars());
            v.setPadding(systemBars.left, systemBars.top, systemBars.right, systemBars.bottom);
            return insets;
        });
    }

    // Reload models if flag is set
    @Override
    protected void onResume() {
        super.onResume();
        if (reloadModelsFlag) reloadModels();
    }

    // Fetch all models from API and update models dropdown accordingly
    protected void reloadModels() {
        showLoadingScreen("Fetching models...");
        if (!initializeApiClient()) {
            return;
        }
        // Get models request future
        CompletableFuture<JsonResponse> future = apiClient.getModels();
        future.thenAcceptAsync(response -> {
            // Read response
            JSONArray responseArray = response.jsonArray;
            // Clear dropdown menu and add static entry
            spinnerItems.clear();
            spinnerItems.add("-- Select a model --");
            // Add all models to dropdown menu
            for (int i = 0; i < responseArray.length(); i++) {
                try {
                    String modelName = responseArray.getJSONObject(i).getString("Name");
                    String initCommands = responseArray.getJSONObject(i).getString("Init_commands");
                    String successKeyword = responseArray.getJSONObject(i).getString("Success_keywords");
                    modelCommandsMap.put(modelName, new ModelCommands(successKeyword, initCommands));
                    spinnerItems.add(modelName);
                } catch (JSONException e) {
                    throw new RuntimeException(e);
                }
            }
            // Notify adapter about update and select default entry
            runOnUiThread(() -> {
                adapter.notifyDataSetChanged();
                modelSpinner.setSelection(0);
                hideLoadingScreen();
            });
            reloadModelsFlag = false;
        }, executor).exceptionally(e -> {
            showCriticalMsg(this, "Failed to fetch models", "Please verify your internet connectivity");
            return null;
        });
        checkPermissions();
    }



    private boolean initializeApiClient(){
        // Load api key and the owner name
        SharedPreferences sharedPreferences = getSharedPreferences("DATA", Context.MODE_PRIVATE);
        String apikey = sharedPreferences.getString("api-key",null);
        String apikeyowner = sharedPreferences.getString("key-owner",null);

        // Display api key owner name
        keyOwnerText.setText("API Key Owner: " + apikeyowner);

        // Switch to LoginActivity if api key is not set
        if (apikey == null){
            startActivity(new Intent(MainActivity.this,LoginActivity.class));
            return false;
        }

        // Initialize api client
        apiClient = new apiClient(this, apikey);
        return true;
    }

    // Request SEND_SMS permissions if not already granted
    private boolean checkPermissions() {
        if (ContextCompat.checkSelfPermission(this, Manifest.permission.SEND_SMS) != PackageManager.PERMISSION_GRANTED) {
            showMessage(innerLayout, "Please go into settings and give this app SEND_SMS permission");
            return false;
        }
        return true;
    }

    // Build barcode/QR code scanner
    private GmsBarcodeScanner ScanBuilder() {
        GmsBarcodeScannerOptions options =
                new GmsBarcodeScannerOptions.Builder().setBarcodeFormats(Barcode.FORMAT_CODE_128,Barcode.FORMAT_QR_CODE).enableAutoZoom().build();
        return GmsBarcodeScanning.getClient(this, options);
    }

    // Launch scanner and fill in relevant text box
    private void launchBarcodeScanner(EditText fieldToUpdate) {
        scanner.startScan()
                .addOnSuccessListener(barcode -> fieldToUpdate.setText(barcode.getRawValue()))
                .addOnFailureListener(e -> showMessage(innerLayout, e.getMessage()));
    }

    private void provision() {
        // Validate model selection
        String model = modelSpinner.getSelectedItem().toString();
        if (!InputValidationHelper.isValidModel(model)){
            showMessage(innerLayout, "Invalid model selected");
            return;
        }

        // Validate phone number
        String phoneNumber = phonenumberField.getText().toString().replaceAll("[^\\d+]", "");
        if (!InputValidationHelper.isValidPhoneNumber(phoneNumber)){
            showMessage(innerLayout, "Invalid Phone Number");
            return;
        }

        // Validate validate id field
        String id = idField.getText().toString();
        if (!InputValidationHelper.isValidId(id)){
            showMessage(innerLayout, "Invalid ID entered. ID must be a general ID of 12 digits or a IMEI of 15 digits");
            return;
        }

        // Validate name field
        String name = nameField.getText().toString();
        if (!InputValidationHelper.isValidName(name)){
            showMessage(innerLayout, "Invalid name entered. The name must be 1-32 characters");
            return;
        }

        boolean validate = validateProvisioning.isChecked();


        showLoadingScreen("Starting the provisioning process");
        TrackerDetails td = new TrackerDetails(id, name, phoneNumber, model, true);

        try{
            // Register Tracker
            setLoadingText("Registering tracker by API and SMS...");
            // Get resgistrion future, and define error handling
            CompletableFuture<JsonResponse> registrationFuture = apiClient.registerTracker(td)
                    .orTimeout(2, TimeUnit.SECONDS)
                    .exceptionally(e -> {
                        if (e instanceof java.util.concurrent.TimeoutException){
                            throw new CompletionException("Timed out while registering tracker with API",e);
                        }
                        else {
                            throw new CompletionException("Failed to register tracker with API",e);
                        }
                    });
            // Get sms provisioning future and define error handling
            CompletableFuture<Void> smsFuture = startSMSProvisioning(td,validate)
                    .orTimeout(360, TimeUnit.SECONDS)
                    .exceptionally(e -> {
                        if (e instanceof java.util.concurrent.TimeoutException){
                            throw new CompletionException("Timed out waiting for SMS response from tracker",e);
                        }
                        else {
                            throw new CompletionException("Failed to SMS provision tracker",e);
                        }
                    });
            // Wait for registration future, and write status to screen
            registrationFuture.thenCompose(registrationRes -> {
                switch (registrationRes.statusCode){
                    case 200:
                        setLoadingText("Tracker has been successfully registered on the API.\nWaiting for SMS provisioning to complete...");
                        break;
                    case 409:
                        setLoadingText("You already have a tracker registered with the same id or name.\nWaiting for SMS provisioning to complete...");
                        break;
                    case 403:
                        return CompletableFuture.failedFuture(new Exception("Failed: Tracker with the same id or name is already registered with another user. "));
                }
                return smsFuture;
            // Wait for sms provisioning future and write status to screen
            }).thenCompose(v -> {
                if (validate){
                    setLoadingText("Waiting for the tracker to connect to the backend service...");
                    return apiClient.waitUntilTrackerConnected(td.id).orTimeout(60,TimeUnit.SECONDS)
                            .exceptionally(e -> {
                                if (e instanceof java.util.concurrent.TimeoutException){
                                    throw new CompletionException("Timed out waiting for tracker to connect",e);
                                }
                                else {
                                    throw new CompletionException("Failed waiting for tracker to connect",e);
                                }
                            });
                }
                else {
                    return CompletableFuture.completedFuture(null);
                }
            })
            // Wait for tracker to validate if validate boolean is set
            .thenAccept(result -> {
                runOnUiThread(() -> hideLoadingScreen());
                showMessage(innerLayout,"Tracker was successfully provisioned");
            })
            // Print error on exception
            .exceptionally(e -> {
                String errorMessage = e.getCause().getMessage();
                runOnUiThread(() -> hideLoadingScreen());
                showMessage(innerLayout,errorMessage);
                return null;
            });
        }
        catch (Exception e){
            System.out.println("Something went wrong");
        }
    }



   public CompletableFuture<Void> startSMSProvisioning(TrackerDetails td,boolean waitForResponse) {
        // fetch and validate commands for selected tracker
        ModelCommands modelCommands = modelCommandsMap.get(td.model);
        if (modelCommands == null || modelCommands.init_commands == null || modelCommands.init_commands.length < 1) {
            runOnUiThread(() -> showMessage(innerLayout, "No commands registered for selected tracker model"));
            hideLoadingScreen();
            return CompletableFuture.failedFuture(new Exception("No commands for tracker"));
        } else {
            // Initiate smscontroller object, start it and return its future
            SmsController smsController = new SmsController(getApplicationContext(),
                    td.phoneNumber,
                    td.id,
                    modelCommands,
                    waitForResponse);
            if (checkPermissions()) {
                return smsController.start();
            }
            else {
                return CompletableFuture.failedFuture(new Exception("Permissions not granted"));
            }
        }
    }

    // Block user interaction, and show loading layout and text
    private void showLoadingScreen(String text){
        runOnUiThread(() -> {
                blockUserInteractions(innerLayout, true);
                blockUserInteractions(settingsBtn, true);
                loadingText.setText(text);
                loadingLayout.setVisibility(View.VISIBLE);
                });
    }

    // Enable user interaction, and hide loading layout and text
    private void hideLoadingScreen(){
        runOnUiThread(() -> {
            blockUserInteractions(innerLayout, false);
            blockUserInteractions(settingsBtn, false);
            loadingText.setText("");
            loadingLayout.setVisibility(View.GONE);
        });
    }

    private void setLoadingText(String text){
        runOnUiThread(() -> loadingText.setText(text));
    }

}

// Helper for performing field validations
class InputValidationHelper{
    public static boolean isValidModel(String model){
        return !model.startsWith("-");
    }
    public static boolean isValidPhoneNumber(String phoneNumber){
        return phoneNumber.matches("^[0-9]{8}$") || phoneNumber.matches("^\\+45[0-9]{8}$");
    }

    public static boolean isValidId(String id){
        return id.matches("[0-9]+") && (id.length() == 12 || id.length() == 15);
    }
    public static boolean isValidName(String name){
        return name.length() >= 1 && name.length() <= 32;
    }
}


class ModelCommands{
    public String[] success_keyword;
    public String[] init_commands;
    public ModelCommands(String success_keyword,String init_commands){
        this.success_keyword = success_keyword.split(";");
        this.init_commands = init_commands.split(";");
    }
}

class TrackerDetails{
    public String id,name,phoneNumber,model;
    public boolean enabled;
    public TrackerDetails(String id, String name, String phoneNumber, String model, boolean enabled){
            this.id = id.trim();
            this.name = name.trim();
            this.phoneNumber = phoneNumber;
            this.model = model;
            this.enabled = enabled;
    }

    @NonNull
    public String toString() {
        Gson gson = new Gson();
        return gson.toJson(this);
    }
    public JSONObject toJsonObject() throws JSONException {
        return new JSONObject(this.toString());
    }
}




