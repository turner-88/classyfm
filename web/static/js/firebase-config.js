// Firebase bootstrap for the Connect chat. Loaded (deferred, in <head>) after the vendored
// firebase-*-compat SDKs and before connect.js, so the global `firebase` exists here and the
// window.fb* singletons exist by the time connect.js runs. In <head> so Turbo preserves it
// across navigations (it initializes once per hard load).
//
// The Connect chat shares the mobile app's Realtime Database (project classyfm-dd873):
// prod messages live at /chats, dev/staging at /chatsdev. See connect.js for the reader.
//
// This whole config is PUBLIC and safe to ship in client JS — access is governed by the
// database security rules, not by keeping these values secret.
(function () {
  "use strict";

  // === Firebase web config ===================================================
  // Fill apiKey / appId / messagingSenderId from the Firebase console:
  //   Project settings -> General -> "Your apps" (Web app) -> SDK setup and configuration.
  // Confirm databaseURL there too (old 2019-era projects use <id>.firebaseio.com; newer
  // regional instances use <id>-default-rtdb.<region>.firebasedatabase.app).
  var firebaseConfig = {
    apiKey: "AIzaSyDgNynZd00fmH5Lz9vQo77n5JE-o_TJZhQ",
    authDomain: "classyfm-dd873.firebaseapp.com",
    databaseURL: "https://classyfm-dd873.firebaseio.com",
    projectId: "classyfm-dd873",
    storageBucket: "classyfm-dd873.firebasestorage.app",
    messagingSenderId: "776994593728",
    appId: "1:776994593728:web:ae19092a6efd9c52620625",
  };

  // Prod hosts write to /chats; everything else (localhost, previews) uses /chatsdev so
  // local testing never pollutes the live room. Add the real production domain(s) here.
  var PROD_HOSTS = ["classyfm.remorac.com", "classyfm.co.id", "www.classyfm.co.id"];
  var chatsPath = PROD_HOSTS.indexOf(location.hostname) !== -1 ? "/chats" : "/chatsdev";

  if (typeof firebase === "undefined") {
    // The SDK failed to load (blocked/offline). connect.js guards on window.fbDatabase.
    console.warn("[connect] Firebase SDK not loaded; chat disabled.");
    return;
  }
  if (firebaseConfig.apiKey.indexOf("FILL_ME") === 0) {
    console.warn("[connect] Firebase config not filled in (firebase-config.js); chat disabled.");
    return;
  }

  // initializeApp is idempotent-guarded: Turbo re-executes body scripts on every in-site
  // navigation, but firebase-config.js loads in <head> without defer and runs once per hard
  // load. The guard is belt-and-suspenders against a double include.
  if (!firebase.apps.length) {
    firebase.initializeApp(firebaseConfig);
  }

  window.fbAuth = firebase.auth();
  window.fbDatabase = firebase.database();
  window.fbChatsPath = chatsPath;
})();
