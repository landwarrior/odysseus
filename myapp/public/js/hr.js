sumPreCost();
$('.datepicker').datepicker({
    format: 'yyyy/mm/dd',
    language: 'ja',
    todayBtn: 'linked',
    todayHighlight: true,
});
$('#hr-prices').on('click', '.row-delete', function() {
  $(this).parent().parent().remove();
  updateName();
  sumPreCost();
});
$('.datepicker').datepicker({
    format: 'yyyy/mm/dd',
    language: 'ja',
    todayBtn: 'linked',
    todayHighlight: true,
});
$('#hr-prices').on('blur', '.price', function() {
  let price = $(this).val().replace(/[^0-9-.]/g, '');
  $(this).parent().find('input[type=hidden].price').val(price);
  sumPreCost();
});
function updateName() {
  let num = 0;
  $('#hr-prices tbody tr').each(function() {
    $(this).find('select.form-control').attr('name', 'prices[' + num + '][role_id]');
    $(this).find('input[type=hidden].price').attr('name', 'prices[' + num + '][price]');
    $(this).find('input.from_date').attr('name', 'prices[' + num + '][from_date]');
    num++;
  });
}
function addRow() {
  $('#cloneTr tr').clone().appendTo('#hr-prices tbody');
  updateName();
  $('.datepicker').datepicker({
      format: 'yyyy/mm/dd',
      language: 'ja',
      todayBtn: 'linked',
      todayHighlight: true,
  });
}
function sumPreCost() {
  let total = 0;
  $('#hr-prices tbody tr').each(function() {
    let pre_cost = Number($(this).find('input[type=hidden].pre_cost').val());
    if (!isNaN(pre_cost)) {
      total += pre_cost;
    }
  });
  $('.total-pre-cost').text('￥ ' + total.toLocaleString());
}

$('#conform-delete').on('input', function() {
  let inputText = $(this).val();
  let hrCd = $('#hr_cd').val();
  if (hrCd == inputText) {
    $('#delete-execute').attr('disabled', false);
  } else {
    $('#delete-execute').attr('disabled', true);
  }
});
