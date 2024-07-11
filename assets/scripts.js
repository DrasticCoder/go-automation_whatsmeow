$(document).ready(function() {
    // Function to check login status and update UI accordingly
    function checkLoginStatus() {
        $.ajax({
            url: "/loginstatus",
            type: "GET",
            success: function(response) {
                if (response.logged_in) {
                    $("#qr-container").hide();
                    $("#form-container").show();
                    $("#logout-btn").show();
                } else {
                    $("#qr-container").show();
                    $("#form-container").hide();
                    $("#logout-btn").hide();
                }
            },
            error: function(error) {
                console.error("Error fetching login status:", error);
            }
        });
    }

    // Check login status on page load
    checkLoginStatus();

    // Logout button functionality
    $("#logout-btn").click(function() {
        $.ajax({
            url: "/logout",
            type: "GET",
            success: function(response) {
                $("#form-container").hide();
                $("#logout-btn").hide();
                $("#qr-container").show();
                checkLoginStatus(); // Check and update login status after logout
            },
            error: function(error) {
                console.error("Error logging out:", error);
            }
        });
    });
});
