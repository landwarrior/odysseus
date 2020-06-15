<?php

namespace App\Http\Controllers\Auth;

use App\Http\Controllers\Controller;
use App\Providers\RouteServiceProvider;
use App\MstHr;
use Illuminate\Foundation\Auth\RegistersUsers;
use Illuminate\Support\Facades\Hash;
use Illuminate\Support\Facades\Validator;

class RegisterController extends Controller
{
    /*
    |--------------------------------------------------------------------------
    | Register Controller
    |--------------------------------------------------------------------------
    |
    | This controller handles the registration of new users as well as their
    | validation and creation. By default this controller uses a trait to
    | provide this functionality without requiring any additional code.
    |
    */

    use RegistersUsers;

    /**
     * Where to redirect users after registration.
     *
     * @var string
     */
    protected $redirectTo = RouteServiceProvider::HOME;

    /**
     * Create a new controller instance.
     *
     * @return void
     */
    public function __construct()
    {
        $this->middleware('guest');
    }

    /**
     * Get a validator for an incoming registration request.
     *
     * @param  array  $data
     * @return \Illuminate\Contracts\Validation\Validator
     */
    protected function validator(array $data)
    {
        return Validator::make($data, [
            'hr_cd' => ['required', 'string', 'max:16', 'unique:mst_hr'],
            'user_name' => ['required', 'string', 'max:128'],
            'name_kana' => ['string', 'max:255'],
            'password' => ['required', 'string', 'min:8', 'confirmed'],
            'remarks' => ['string', 'max:256'],
        ]);
    }

    /**
     * Create a new user instance after a valid registration.
     *
     * @param  array  $data
     * @return \App\MstHr
     */
    protected function create(array $data)
    {
        $hr = new MstHr();
        $hr->hr_cd = $data['hr_cd'];
        $hr->user_name = $data['user_name'];
        $hr->name_kana = $data['name_kana'];
        $hr->password = Hash::make($data['password']);
        $hr->is_admin = 0;
        $hr->remarks = $data['remarks'];
        $hr->save();
        return $hr;
    }
}
