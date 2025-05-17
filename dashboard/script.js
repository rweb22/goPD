document.addEventListener("DOMContentLoaded", function() {

    let today = new Date();
    let year = today.getFullYear();
    let month = String(today.getMonth() + 1).padStart(2, '0'); // Ensure two-digit month
    let day = String(today.getDate()).padStart(2, '0'); // Ensure two-digit day
    let date = `${year}-${month}-${day}`; 
    let stock = "BANKNIFTY";
    let pdi = "5";
    let pdc = "PD";
    let pdData = {}
    let lastAlarm = null;
    let pdt = 2;
    let alarmON = true;
    // Initialize Bootstrap Datepicker
    $("#datepicker").datepicker({
        format: "yyyy-mm-dd",
        autoclose: true,
        todayHighlight: true
    }).datepicker("setDate", date);

    const redAlert = document.getElementById("redAlert");
    const greenAlert = document.getElementById("greenAlert");
    const notification = document.getElementById("notification");

    // Function to Fetch Stock Data
    async function fetchStockData() {
        try {
            let response = await fetch(`/pd`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ date, stock, pdi })
            });

            if (!response.ok) {
                throw new Error("Failed to fetch stock data");
            }

            pdData = await response.json();
	    plotGraph();
        } catch (error) {
            console.error("Error fetching stock data:", error);
            return null;
        }
    }

    function plotGraph() {
	    let times = new Set([...Object.keys(pdData["stock_price"]), ...Object.keys(pdData[pdc])]);
	    let sortedTimes = Array.from(times).sort();

	    let yA = sortedTimes.map(t => pdData["stock_price"][t] !== undefined ? pdData["stock_price"][t] : null);
	    let yB = sortedTimes.map(t => pdData[pdc][t] !== undefined ? pdData[pdc][t] : null);

	    let traceA = {
		    x: sortedTimes,
		    y: yA,
		    mode: 'lines+markers',
		    name: 'Stock Price',
		    line: {color: 'blue'},
		    yaxis: 'y1'
	    };

	    let traceB = {
		    x: sortedTimes,
		    y: yB,
		    mode: 'lines+markers',
		    name: 'PD',
		    line: {color: 'red'},
		    yaxis: 'y2'
	    }

	    let layout = {
		    title: 'Time vs Stock Price vs PD',
		    xaxis: {title: 'Time (HH:MM)'},
		    yaxis: {
			    title: 'Stock Price',
			    side: 'left',
			    showgrid: false,
			    zeroline: false
		    },
		    yaxis2: {
			    title: "PD",
			    overlaying: 'y',
			    side: 'right',
			    showgrid: false,
			    zeroline: false
		    }
	    };

	    Plotly.newPlot('chart', [traceA, traceB], layout);

	    pd = pdData["PD"];
	    price = pdData["stock_price"]

	    if (pd[-1] > pdt * 1000000000 || pd[-1] < pdt * (-1000000000)) {
		    triggerAlarm("greenAlert", `PD has crossed the ${pdt} Billion mark`, "success");
	    }

	    if (((pd[-1]-pd[-2])*(price[-1]-price[-2]) > 0) && ((pd[-2]-pd[-3])*(price[-2]-price[-3]) > 0)) {
		    triggerAlarm("redAlert", `PD data anomaly detected`, "danger");
	    }
    }

    function triggerAlarm(type, message, color) {
	    if ((lastAlarm === type) || !alarmON) return;
	    lastAlarm = type;

	    if (type === "redAlert") {
		    redAlert.play();
	    } else if (type === "greenAlert") {
		    greenAlert.play();
	    }

	    showToast(message, color);
    }

    function showToast(message, color) {
	    const toastId = "toast- " + Date.now();
	    const toast = document.createElement("div");
	    toast.className = `toast align-items-center text-white bg-${color} border-0`;
	    toast.setAttribute = ("role", "alert");
	    toast.setAttribute = ("aria-live", "assertive");
	    toast.setAttribute = ("aria-atomic", "true");
	    toast.setAttribute = ("id", toastId);
	    toast.innerHTML = `
      		<div class="d-flex">
        		<div class="toast-body">
          			${message}
        		</div>
        		<button type="button" class="btn-close btn-close-white me-2 m-auto" data-bs-dismiss="toast" aria-label="Close"></button>
      		</div>
    		`;
	    notification.appendChild(toast);

	    const bsToast = new bootstrap.Toast(toast, { autohide: false });
	    bsToast.show();

	    toast.addEventListener('hidden.bs.toast', () => {
		    if (toast.parentElement) toast.remove();
		    lastAlarm = null;
	    });
    }

    // "GO" Button Click Event
    document.getElementById("goButton").addEventListener("click", async function() {
        date = document.getElementById("datepicker").value;
        stock = document.getElementById("optionSelector").value;
        pdi = document.getElementById("oddNumberSelector").value.toString();

        console.log("Selected Date:", date);
        console.log("Selected Option:", stock);
        console.log("Selected Number:", pdi);

        fetchStockData();
    });

    document.querySelector('#pd-tabs').addEventListener('shown.bs.tab', function(event) {
        const newActiveTabId = event.target.id;
	if (newActiveTabId === 'oi-tab') {
		pdc = "PD";
	} else {
		pdc = "PDV";
	}
	plotGraph();
    });

    document.getElementById('applyBtn').addEventListener('click', () => {
	    const pdThreshold = parseFloat(document.getElementById('thresholdInput').value);
	    if (!isNaN(pdThreshold) && pdThreshold > 0){
		    pdt = pdThreshold;
		    alarmON = true;
	    } else {
		    alert("Alarm turned OFF. Please enter a valid positive number.");
		    alarmON = false;
	    }
    });

    // Load default graph on page load
    document.getElementById("goButton").click();

    setInterval(() => {
	    fetchStockData();
    }, 60000);
});

