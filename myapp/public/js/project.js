sumPreCost();
$('.datepicker').datepicker({
    format: 'yyyy/mm/dd',
    language: 'ja',
});
$('#project-details').on('click', '.row-delete', function() {
  $(this).parent().parent().remove();
  updateName();
  sumPreCost();
});
$('#project-details').on('blur', '.pre_cost', function() {
  let pre_cost = $(this).val().replace(/[^0-9-.]/g, '');
  $(this).parent().find('input[type=hidden].pre_cost').val(pre_cost);
  sumPreCost();
});
function updateName() {
  let num = 0;
  $('#project-details tbody tr').each(function() {
    $(this).find('select.form-control').attr('name', 'details[' + num + '][process_id]');
    $(this).find('input.from_date').attr('name', 'details[' + num + '][from_date]');
    $(this).find('input.to_date').attr('name', 'details[' + num + '][to_date]');
    $(this).find('input.man_per_day').attr('name', 'details[' + num + '][man_per_day]');
    $(this).find('input[type=hidden].pre_cost').attr('name', 'details[' + num + '][pre_cost]');
    num++;
  });
}
function addRow() {
  $('#cloneTr tr').clone().appendTo('#project-details tbody');
  updateName();
  $('.datepicker').datepicker({
      format: 'yyyy/mm/dd',
      language: 'ja',
  });
}
function sumPreCost() {
  let total = 0;
  $('#project-details tbody tr').each(function() {
    let pre_cost = Number($(this).find('input[type=hidden].pre_cost').val());
    if (!isNaN(pre_cost)) {
      total += pre_cost;
    }
  });
  $('.total-pre-cost').text('￥ ' + total.toLocaleString());
}

$('#conform-delete').on('input', function() {
  let inputText = $(this).val();
  let projectNo = $('#project_no').val();
  if (projectNo == inputText) {
    $('#delete-execute').attr('disabled', false);
  } else {
    $('#delete-execute').attr('disabled', true);
  }
});
