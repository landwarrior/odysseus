<div class="container">
  <div class="row">
    <div class="col-md-12">
      @include('message')
      <div class="card">
        <div class="card-header">{{ __('messages.hr.create') }}</div>
        <div class="card-body">
          @if($target == 'store')
          <form action="/hr" method="post">
          @elseif($target == 'update')
          <form action="/hr/{{ $human->hr_cd }}" method="post">
            <input type="hidden" name="_method" value="PUT">
          @endif
            @csrf
            <div class="row">
              <div class="col-lg-3">
                <div class="form-group">
                  @if ($target == 'store')
                  <div class="form-label-group">
                    <input type="text" id="hr_cd" class="form-control @error('hr_cd') is-invalid @enderror" name="hr_cd" value="{{ old('hr_cd') }}" required placeholder="{{ __('messages.hr.hr_cd') }}">
                    <label for="hr_cd">{{ __('messages.hr.hr_cd') }}</label>
                  </div>
                  @else
                  <div class="form-label-group">
                    <input type="text" id="hr_cd" class="form-control" name="hr_cd" value="{{ $human->hr_cd }}" disabled placeholder="{{ __('messages.hr.hr_cd') }}">
                    <label for="hr_cd">{{ __('messages.hr.hr_cd') }}</label>
                  </div>
                  @endif
                </div>
              </div>
              <div class="col-lg-auto">
                <div class="text-center" style="margin: 0">{{ __('messages.hr.is_admin') }}</div>
                <input type="hidden" name="is_admin" value="0">
                <input type="checkbox" class="form-control" id="is_admin" name="is_admin" value="1" style="margin: 0 auto" @if(old('is_admin', $human->is_admin)) checked @endif>
              </div>
            </div>
            <div class="row">
              <div class="col-lg-6">
                <div class="form-group">
                  <div class="form-label-group">
                    <input type="text" id="user_name" class="form-control @error('user_name') is-invalid @enderror" name="user_name" value="{{ old('user_name', $human->user_name) }}" placeholder="{{ __('messages.hr.user_name') }}">
                    <label for="user_name">{{ __('messages.hr.user_name') }}</label>
                  </div>
                </div>
              </div>
              <div class="col-lg-6">
                <div class="form-group">
                  <div class="form-label-group">
                    <input type="text" id="name_kana" class="form-control @error('name_kana') is-invalid @enderror" name="name_kana" value="{{ old('name_kana', $human->name_kana) }}" placeholder="{{ __('messages.hr.name_kana') }}">
                    <label for="name_kana">{{ __('messages.hr.name_kana') }}</label>
                  </div>
                </div>
              </div>
            </div>
            <div class="row">
              <div class="col-12" style="padding-bottom: 20px">
                <textarea class="form-control" name="remarks" placeholder="{{ __('messages.hr.remarks') }}">{{ old('remarks', $human->remarks) }}</textarea>
              </div>
            </div>

            <div class="row">
              <div class="col-12">
                <table id="hr-prices" class="table">
                  <thead>
                    <tr class="text-center">
                      <th scope="col" style="width:150px">{{ __('messages.hr.role') }}</th>
                      <th scope="col">{{ __('messages.hr.price') }}</th>
                      <th scope="col">{{ __('messages.hr.from') }}</th>
                      <th scope="col" style="width:65px">{{ __('messages.delete') }}</th>
                    </tr>
                  </thead>
                  <tbody>
                    @foreach(old('prices', $prices) as $price)
                    <tr>
                      <td class="form-group">
                        <select class="form-control @error('prices.'.$loop->index.'.role_id') is-invalid @enderror" name="prices[{{ $loop->index }}][role_id]">
                          @for($i = 0; $i < count($roles); $i++)
                          <option value="{{ $roles[$i]->role_id }}"
                                  @if ($roles[$i]->role_id == old('prices.'.$loop->index.'.role_id', !empty($price->role_id) ? $price->role_id : 0)) selected @endif>{{ $roles[$i]->name }}</option>
                          @endfor
                        </select>
                      </td>
                      <td>
                        <input type="text" class="form-control text-right price numFmt @error('prices.'.$loop->index.'.price') is-invalid @enderror"
                               value="{{ old('prices.'.$loop->index.'.price', !empty($price->price) ? $price->price : '') }}">
                        <input type="hidden" class="price" name="prices[{{ $loop->index }}][price]"
                               value="{{ old('prices.'.$loop->index.'.price', !empty($price->price) ? $price->price : '') }}">
                      </td>
                      <td>
                        <input type="text" class="form-control text-center from_date datepicker @error('prices.'.$loop->index.'.from_date') is-invalid @enderror"
                               name="prices[{{ $loop->index }}][from_date]"
                               value="{{ old('prices.'.$loop->index.'.from_date', !empty($price->from_date) ? $price->from_date->format('Y/m/d') : '') }}">
                      </td>
                      <td>
                        <button type="button" class="btn btn-danger row-delete">
                          <svg class="bi bi-trash" width="1em" height="1em" viewBox="0 0 16 16" fill="currentColor" xmlns="http://www.w3.org/2000/svg">
                            <path d="M5.5 5.5A.5.5 0 0 1 6 6v6a.5.5 0 0 1-1 0V6a.5.5 0 0 1 .5-.5zm2.5 0a.5.5 0 0 1 .5.5v6a.5.5 0 0 1-1 0V6a.5.5 0 0 1 .5-.5zm3 .5a.5.5 0 0 0-1 0v6a.5.5 0 0 0 1 0V6z"/>
                            <path fill-rule="evenodd" d="M14.5 3a1 1 0 0 1-1 1H13v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V4h-.5a1 1 0 0 1-1-1V2a1 1 0 0 1 1-1H6a1 1 0 0 1 1-1h2a1 1 0 0 1 1 1h3.5a1 1 0 0 1 1 1v1zM4.118 4L4 4.059V13a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1V4.059L11.882 4H4.118zM2.5 3V2h11v1h-11z"/>
                          </svg>
                        </button>
                      </td>
                    </tr>
                    @endforeach
                  </tbody>
                  <tfoot>
                    <tr>
                      <td colspan="4" class="text-right">
                        <button type="button" class="btn btn-primary" onclick="addRow()">{{ __('messages.add_row') }}</button>
                      </td>
                    </tr>
                  </tfoot>
                </table>
              </div>
            </div>

            <div class="row">
              <div class="col-6">
                <button type="submit" class="btn btn-lg btn-outline-primary">{{ __('messages.register') }}</button>
              </div>
              <div class="col-6 text-right">
                <a href="/hr" class="btn btn-lg btn-outline-secondary">{{ __('messages.back') }}</a>
              </div>
            </div>
          </form>
        </div>
        @if($target == 'update')
        <div class="card-header alert-danger text-center">{{ __('messages.danger_zone') }}</div>
        <div class="card-body">
          <form action="/hr/{{ $human->hr_cd }}" method="post" name="delete">
            @csrf
            <input type="hidden" name="_method" value="DELETE">
            <button type="button" id="delete-btn" class="btn btn-lg btn-block btn-danger" data-toggle="modal" data-target="#delete-modal">{{ __('messages.delete') }}</button>
          </form>
        </div>
        @endif
      </div>
    </div>
  </div>
</div>
<table id="cloneTr" style="display: none">
  <tr>
    <td class="form-group">
      <select class="form-control">
        @foreach($roles as $role)
        <option value="{{ $role->role_id }}">{{ $role->name }}</option>
        @endforeach
      </select>
    </td>
    <td>
      <input type="text" class="form-control text-right price numFmt">
      <input type="hidden" class="price">
    </td>
    <td>
      <input type="text" class="form-control text-center from_date datepicker">
    </td>
    <td>
      <button type="button" class="btn btn-danger row-delete">
        <svg class="bi bi-trash" width="1em" height="1em" viewBox="0 0 16 16" fill="currentColor" xmlns="http://www.w3.org/2000/svg">
          <path d="M5.5 5.5A.5.5 0 0 1 6 6v6a.5.5 0 0 1-1 0V6a.5.5 0 0 1 .5-.5zm2.5 0a.5.5 0 0 1 .5.5v6a.5.5 0 0 1-1 0V6a.5.5 0 0 1 .5-.5zm3 .5a.5.5 0 0 0-1 0v6a.5.5 0 0 0 1 0V6z"/>
          <path fill-rule="evenodd" d="M14.5 3a1 1 0 0 1-1 1H13v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V4h-.5a1 1 0 0 1-1-1V2a1 1 0 0 1 1-1H6a1 1 0 0 1 1-1h2a1 1 0 0 1 1 1h3.5a1 1 0 0 1 1 1v1zM4.118 4L4 4.059V13a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1V4.059L11.882 4H4.118zM2.5 3V2h11v1h-11z"/>
        </svg>
      </button>
    </td>
  </tr>
</table>
<div class="modal fade" id="delete-modal" tabindex="-1" role="dialog" aria-labelledby="delete-modal-label" aria-hidden="true">
  <div class="modal-dialog" role="document">
    <div class="modal-content">
      <div class="modal-header alert-danger">
        <h5 class="modal-title">{{ __('messages.delete') }}</h5>
        <button type="button" class="close" data-dismiss="modal" aria-label="Close">
          <span aria-hidden="true">&times;</span>
        </button>
      </div>
      <div class="modal-body">
        {{ __('messages.hr.del_confirm') }}
        <div class="form-group">
          <div class="form-label-group">
            <input type="text" class="form-control" id="conform-delete" placeholder="{{ __('messages.hr.hr_cd') }}">
            <label for="conform-delete">{{ __('messages.hr.hr_cd') }}</label>
          </div>
        </div>
      </div>
      <div class="modal-footer">
        <button type="button" class="btn btn-secondary" data-dismiss="modal">{{ __('messages.close') }}</button>
        <button type="button" id="delete-execute" class="btn btn-danger" disabled onclick="document.delete.submit();">{{ __('messages.delete') }}</button>
      </div>
    </div>
  </div>
</div>
@include('hr.datepicker')
