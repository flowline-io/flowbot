document.getElementById("data").addEventListener("change", function (e) {
    try {
        const myJSON = JSON.parse(e.target.value);
        const formatter = new JSONFormatter(myJSON);
        document.getElementById("view").innerHTML = "";
        document.getElementById("view").appendChild(formatter.render());
    }catch (e) {
        UIkit.notification(e);
    }
})
