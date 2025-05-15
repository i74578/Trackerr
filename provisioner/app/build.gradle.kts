plugins {
    alias(libs.plugins.android.application)
}

android {
    namespace = "com.trackerr.provisioner"
    compileSdk = 35

    defaultConfig {
        applicationId = "com.trackerr.provisioner"
        minSdkVersion(rootProject.extra["defaultMinSdkVersion"] as Int)
        targetSdk = 35
        versionCode = 1
        versionName = "1.0"

        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
    }

    buildTypes {
        release {
            isMinifyEnabled = false
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
        }
    }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_11
        targetCompatibility = JavaVersion.VERSION_11
    }
}

dependencies {
    implementation(libs.volley)
    implementation(libs.play.services.code.scanner)
    implementation(libs.gson)
    implementation(libs.appcompat)
    implementation(libs.material)
    implementation(libs.activity)
    implementation(libs.constraintlayout)
    androidTestImplementation(libs.espresso.core)
}