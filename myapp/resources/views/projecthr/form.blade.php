<div class="container">
  <div class="row">
    <div class="col-md-12">
      @include('message')
      <div class="card">
        <div class="card-header">{{ __('messages.projecthr.create') }}</div>
        <div class="card-body">
          <form action="/projecthr/{{ $project->project_no }}" method="post">
            <input type="hidden" name="_method" value="PUT">
            @csrf
            <div class="row">
              <div class="col-lg-3">
                <div class="form-group">
                  <div class="form-label-group">
                    <input type="text" id="project_no" class="form-control" name="project_no" value="{{ $project->project_no }}" disabled placeholder="{{ __('messages.project.no') }}">
                    <label for="project_no">{{ __('messages.project.no') }}</label>
                  </div>
                </div>
              </div>
              <div class="col-lg-3">
                <div class="form-group">
                  <div class="form-label-group">
                    <input type="text" id="order_amount" class="form-control text-right" name="order_amount" value="￥ {{ number_format($project->order_amount) }}" disabled placeholder="{{ __('messages.project.order_amount') }}">
                    <label for="order_amount">{{ __('messages.project.order_amount') }}</label>
                  </div>
                </div>
              </div>
              <div class="col-lg-3">
                <div class="form-group">
                  <div class="form-label-group">
                    <input type="text" id="from_date" class="form-control text-center" name="from_date"
                           value="{{ $project->from_date->format('Y/m/d') }}" disabled placeholder="{{ __('messages.project.from') }}">
                    <label for="from_date">{{ __('messages.project.from') }}</label>
                  </div>
                </div>
              </div>
              <div class="col-lg-3">
                <div class="form-group">
                  <div class="form-label-group">
                    <input type="text" id="to_date" class="form-control text-center" name="to_date"
                           value="{{ $project->to_date->format('Y/m/d') }}" disabled placeholder="{{ __('messages.project.to') }}">
                    <label for="to_date">{{ __('messages.project.to') }}</label>
                  </div>
                </div>
              </div>
              <div class="col-lg-6">
                <div class="form-group">
                  <div class="form-label-group">
                    <input type="text" id="project_name" class="form-control" name="project_name" value="{{ $project->name }}" disabled placeholder="{{ __('messages.project.name') }}">
                    <label for="project_name">{{ __('messages.project.name') }}</label>
                  </div>
                </div>
              </div>
            </div>
            <div class="row">
              <div class="col-12">
                <table class="table">
                  <thead>
                    <tr class="text-center">
                      <th scope="col">{{ __('messages.project.process') }}</th>
                      <th scope="col">{{ __('messages.project.manperday') }}</th>
                      <th scope="col">{{ __('messages.project.precost') }}</th>
                      @foreach($roles as $role)
                      <th scope="col">{{ $role->name }}</th>
                      @endforeach
                    </tr>
                  </thead>
                  <tbody>
                    @foreach($details as $detail)
                    <tr>
                      <td class="form-group">
                        <input type="hidden" name="selects[{{ $loop->index }}][process_id]" value="{{ $detail['process_id'] }}">
                        <span>{{ $detail['process_name'] }}</span>
                      </td>
                      <td class="text-right">
                        <span>{{ $detail['man_per_day'] }}</span>
                      </td>
                      <td class="text-right">
                        <span>￥ {{ number_format($detail['pre_cost']) }}</span>
                      </td>
                      @for($i = 0; $i < count($roles); $i++)
                      <td>
                        <select class="selectpicker form-control" name="selects[{{ $loop->index }}][{{ $roles[$i]->name }}][hrs][]" multiple>
                          @foreach ($hrs as $hr)
                          <option value="{{ $hr->hr_cd }}"
                            @if(isset($detail['selected'][$hr->hr_cd])
                            && $roles[$i]->role_id == $detail['selected'][$hr->hr_cd])
                            selected
                            @endif >{{ $hr->user_name }}</option>
                          @endforeach
                        </select>
                      </td>
                      @endfor
                    </tr>
                    @endforeach
                  </tbody>
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
      </div>
    </div>
  </div>
</div>
@include('projecthr.select')
