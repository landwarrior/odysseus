$('.numFmt').on('focus', function() {
  $(this).val($(this).val().replace(/[^0-9.]/g, ''));
});
$('.numFmt').on('blur', function() {
  let num = Number.parseInt($(this).val());
  if (!isNaN(num)) {
    $(this).val(num.toLocaleString());
  } else {
    $(this).val(0);
  }
});
