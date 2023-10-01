var oTable;

$( document ).ready(function() {
	
	$('.page_menu a').each(function(e) {
		if(window.location.pathname.includes($(this).attr('href'))){
			$(this).attr('class', 'active');
		}
    });
	
	teams = []
	
	$.getJSON("/teams", function(result){
		$.each(result, function(i, field){
			teams.push(field + ":" + i)
		});
		
		teams.sort();
		teams.unshift("Choisis une équipe...")
		
		// select filter
		$('#team-filter').append('<label>&nbsp; Equipe:</label>');
		$('#team-filter').append('<select class="form-control input-sm"  id="sel_team_id"></select>');
		
		for (var ele in teams) {
			var obj = teams[ele].split(":");
			$('#sel_team_id').append('<option value="' + obj[1] + '">' + obj[0] + '</option>');
		}
		// Filter results on select change
		$('#sel_team_id').on('change', function () {
			caption = $(this).find(":selected").text()
			value = $(this).find(":selected").val()
			if(!caption.includes("Choisis")){
				
				if(oTable != null){
					oTable.destroy();
				}else{
					$("#table-container").toggleClass("demo-html-visible");
				}
				
				//console.log( "ready!" );
				oTable = new DataTable('#rakning', {
					responsive: true,
					pageLength: -1,
					lengthChange: false,
					language: {
						url: '//cdn.datatables.net/plug-ins/1.13.6/i18n/fr-FR.json',
					},	
					order: [[0, "asc"]],
					ajax: {
						'url': '/ranking/'+value,
						'dataSrc': ''
					},
					columns: [
						{
							data: 'rank'
						},{
							data: 'teamCaption'
						},{
							data: 'games'
						},{
							data: 'wins',
							render: function(data, clazz, row) {
								return row.wins + " (" + row.winsClear + "/" + row.winsNarrow + ")"
							}
						},{
							data: 'defeats',
							render: function(data, clazz, row) {
								return row.defeats + " (" + row.defeatsClear + "/" + row.defeatsNarrow +")"
							}
						},{
							data: 'setsWon',
							render: function(data, clazz, row) {
								return row.setsWon + "/" + row.setsLost
							}
						},{
							data: 'ballsWon',
							render: function(data, clazz, row) {
								return row.ballsWon + "/" + row.ballsLost
							}
						},{
							data: 'points'
						}
					],
					dom: 'Bfrtip',
					select: true,
					initComplete: function () {
					
					  this.api().rows().every( function ( rowIdx, tableLoop, rowLoop ) {
						var data = this.data();
						
						if (data.teamCaption === caption) {
						  this.select();
						}
					  });
					}
				});
				
				
				
			}
		});
	});
	  
});
