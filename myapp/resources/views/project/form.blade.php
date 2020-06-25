<div class="container">
  <div class="row">
    <div class="col-md-12">
      @include('message')
      <div class="card">
        <div class="card-header">{{ __('messages.project.create') }}</div>
        <div class="card-body">
          @if($target == 'store')
          <form action="/project" method="post">
          @elseif($target == 'update')
          <form action="/project/{{ $project->project_no }}" method="post">
            <input type="hidden" name="_method" value="PUT">
          @endif
            @csrf
            <div class="row">
              <div class="col-lg-3">
                <div class="form-group">
                  @if ($target == 'store')
                  <div class="form-label-group">
                    <input type="text" id="project_no" class="form-control @error('project_no') is-invalid @enderror" name="project_no" value="{{ old('project_no') }}" required placeholder="{{ __('messages.project.no') }}">
                    <label for="project_no">{{ __('messages.project.no') }}</label>
                  </div>
                  @else
                  <div class="form-label-group">
                    <input type="text" id="project_no" class="form-control" name="project_no" value="{{ $project->project_no }}" disabled placeholder="{{ __('messages.project.no') }}">
                    <label for="project_no">{{ __('messages.project.no') }}</label>
                  </div>
                  @endif
                </div>
              </div>
              <div class="col-lg-3">
                <div class="form-group">
                  <div class="form-label-group">
                    <input type="text" id="order_amount" class="form-control text-right numFmt @error('order_amount') is-invalid @enderror" name="order_amount" value="{{ old('order_amount', $project->order_amount) }}" placeholder="{{ __('messages.project.order_amount') }}">
                    <label for="order_amount">{{ __('messages.project.order_amount') }}</label>
                  </div>
                </div>
              </div>
              <div class="col-lg-3">
                <div class="form-group">
                  <div class="form-label-group">
                    <input type="text" id="from_date" class="form-control text-center datepicker @error('from_date') is-invalid @enderror" name="from_date"
                           value="{{ old('from_date', $project->from_date ? $project->from_date->format('Y/m/d') : '') }}" placeholder="{{ __('messages.project.from') }}">
                    <label for="from_date">{{ __('messages.project.from') }}</label>
                  </div>
                </div>
              </div>
              <div class="col-lg-3">
                <div class="form-group">
                  <div class="form-label-group">
                    <input type="text" id="to_date" class="form-control text-center datepicker @error('to_date') is-invalid @enderror" name="to_date"
                           value="{{ old('to_date', $project->to_date ? $project->to_date->format('Y/m/d') : '') }}" placeholder="{{ __('messages.project.to') }}">
                    <label for="to_date">{{ __('messages.project.to') }}</label>
                  </div>
                </div>
              </div>
              <div class="col-lg-6">
                <div class="form-group">
                  <div class="form-label-group">
                    <input type="text" id="project_name" class="form-control @error('project_name') is-invalid @enderror" name="project_name" value="{{ old('project_name', $project->name) }}" required placeholder="{{ __('messages.project.name') }}">
                    <label for="project_name">{{ __('messages.project.name') }}</label>
                  </div>
                </div>
              </div>
            </div>
            <div class="row">
              <div class="col-12">
                <table id="project-details" class="table">
                  <thead>
                    <tr class="text-center">
                      <th scope="col" style="width:150px">{{ __('messages.project.process') }}</th>
                      <th scope="col">{{ __('messages.project.from') }}</th>
                      <th scope="col">{{ __('messages.project.to') }}</th>
                      <th scope="col">{{ __('messages.project.manperday') }}</th>
                      <th scope="col">{{ __('messages.project.precost') }}</th>
                      <th scope="col" style="width:65px">{{ __('messages.delete') }}</th>
                    </tr>
                  </thead>
                  <tbody>
                    @foreach(old('details', $details) as $detail)
                    <tr>
                      <td class="form-group">
                        <select class="form-control @error('details.'.$loop->index.'.process_id') is-invalid @enderror" name="details[{{ $loop->index }}][process_id]">
                          @for($i = 0; $i < count($processes); $i++)
                          <option value="{{ $processes[$i]->process_id }}"
                                  @if ($processes[$i]->process_id == old('details.'.$loop->index.'.process_id', !empty($detail->process_id) ? $detail->process_id : 0)) selected @endif>{{ $processes[$i]->name }}</option>
                          @endfor
                        </select>
                      </td>
                      <td>
                        <input type="text" class="form-control text-center from_date datepicker @error('details.'.$loop->index.'.from_date') is-invalid @enderror"
                               name="details[{{ $loop->index }}][from_date]"
                               value="{{ old('details.'.$loop->index.'.from_date', !empty($detail->from_date) ? $detail->from_date->format('Y/m/d') : '') }}">
                      </td>
                      <td>
                        <input type="text" class="form-control text-center to_date datepicker @error('details.'.$loop->index.'.to_date') is-invalid @enderror"
                               name="details[{{ $loop->index }}][to_date]"
                               value="{{ old('details.'.$loop->index.'.to_date', !empty($detail->to_date) ? $detail->to_date->format('Y/m/d') : '') }}">
                      </td>
                      <td>
                        <input type="number" class="form-control text-right man_per_day @error('details.'.$loop->index.'.man_per_day') is-invalid @enderror"
                               name="details[{{ $loop->index }}][man_per_day]"
                               value="{{ old('details.'.$loop->index.'.man_per_day', !empty($detail->man_per_day) ? $detail->man_per_day : '') }}">
                      </td>
                      <td>
                        <input type="text" class="form-control text-right pre_cost numFmt @error('details.'.$loop->index.'.pre_cost') is-invalid @enderror"
                               value="{{ old('details.'.$loop->index.'.pre_cost', !empty($detail->pre_cost) ? $detail->pre_cost : '') }}">
                        <input type="hidden" class="pre_cost" name="details[{{ $loop->index }}][pre_cost]"
                               value="{{ old('details.'.$loop->index.'.pre_cost', !empty($detail->pre_cost) ? $detail->pre_cost : '') }}">
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
                      <th colspan="4" class="text-right">{{ __('messages.project.total_cost') }}</th>
                      <th class="text-right"><span class="total-pre-cost" style="padding-right: 13px"></span></th>
                      <th></th>
                    </tr>
                    <tr>
                      <td colspan="6" class="text-right">
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
                <a href="/project" class="btn btn-lg btn-outline-secondary">{{ __('messages.back') }}</a>
              </div>
            </div>
          </form>
        </div>
        @if($target == 'update')
        <div class="card-header alert-danger text-center">{{ __('messages.danger_zone') }}</div>
        <div class="card-body">
          <form action="/project/{{ $project->project_no }}" method="post" name="delete">
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
        @foreach($processes as $process)
        <option value="{{ $process->process_id }}">{{ $process->name }}</option>
        @endforeach
      </select>
    </td>
    <td>
      <input type="text" class="form-control text-center from_date datepicker">
    </td>
    <td>
      <input type="text" class="form-control text-center to_date datepicker">
    </td>
    <td>
      <input type="number" class="form-control text-right man_per_day">
    </td>
    <td>
      <input type="text" class="form-control text-right pre_cost numFmt">
      <input type="hidden" class="pre_cost">
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
        {{ __('messages.project.del_confirm') }}
        <div class="form-group">
          <div class="form-label-group">
            <input type="text" class="form-control" id="conform-delete" placeholder="{{ __('messages.project.no') }}">
            <label for="conform-delete">{{ __('messages.project.no') }}</label>
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
@include('project.datepicker')
